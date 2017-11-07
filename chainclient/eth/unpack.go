/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package eth

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"reflect"
)

var (
	errBadBool = errors.New("abi: improperly encoded boolean value")
)

func UnpackEvent(inputs []abi.Argument, v interface{}, output []byte, topics []string) error {
	output = combine(inputs, output, topics)
	return unpack(inputs, v, output)
}

func UnpackTransaction(method abi.Method, v interface{}, hex string) error {
	output := []byte(hex)
	output = txDataFilter(output, method)
	bs := hexutil.MustDecode(string(output))
	return unpack(method.Inputs, v, bs)
}

func txDataFilter(data []byte, method abi.Method) []byte {
	str := common.Bytes2Hex(method.Id())
	length := len(str)
	return []byte("0x" + string(data[2+length:]))
}

// event中indexed field不在data内，而在filterLog的topic内
// topics内容包括eventId以及所有indexed field data
func combine(inputs []abi.Argument, output []byte, topics []string) []byte {
	if len(topics) <= 1 {
		return output
	}
	idxflds := topics[1:]
	j := 0
	k := 0
	var ret [][]byte
	for i := 0; i < len(inputs); i++ {
		if inputs[i].Indexed {
			bs := hexutil.MustDecode(idxflds[j])
			ret = append(ret, bs)
			j++
			continue
		}

		bs := output[k*32 : (k+1)*32]
		ret = append(ret, bs)
		k += 1
	}

	return bytes.Join(ret, []byte{})
}

func unpack(inputs []abi.Argument, v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	var valueOf reflect.Value
	var ok bool
	if valueOf, ok = v.(reflect.Value); !ok {
		valueOf = reflect.ValueOf(v)
	}
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
	)

	if value.Kind() != reflect.Struct {
		return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
	}

	for i := 0; i < len(inputs); i++ {
		marshalledValue, err := toGoType(i, inputs[i], output)

		if err != nil {
			return err
		}

		reflectValue := reflect.ValueOf(marshalledValue)
		for j := 0; j < typ.NumField(); j++ {
			field := typ.Field(j)
			if field.Tag.Get("alias") == inputs[i].Name {
				if err := set(value.Field(j), reflectValue, inputs[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func set(dst, src reflect.Value, output abi.Argument) error {
	dstType := dst.Type()
	srcType := src.Type()

	switch {
	case dstType.AssignableTo(src.Type()):
		dst.Set(src)
	case dstType.Kind() == reflect.Array && srcType.Kind() == reflect.Slice:
		if dst.Len() < output.Type.SliceSize {
			return fmt.Errorf("abi: cannot unmarshal src (len=%d) in to dst (len=%d)", output.Type.SliceSize, dst.Len())
		}
		reflect.Copy(dst, src)
	case dstType.Kind() == reflect.Interface:
		dst.Set(src)
	case dstType.Kind() == reflect.Ptr:
		return set(dst.Elem(), src, output)
	default:
		return fmt.Errorf("abi: cannot unmarshal %v in to %v", src.Type(), dst.Type())
	}
	return nil
}

// toGoType parses the input and casts it to the proper type defined by the ABI
// argument in T.
func toGoType(i int, t abi.Argument, output []byte) (interface{}, error) {
	// we need to treat slices differently
	if (t.Type.IsSlice || t.Type.IsArray) && t.Type.T != abi.BytesTy && t.Type.T != abi.StringTy && t.Type.T != abi.FixedBytesTy && t.Type.T != abi.FunctionTy {
		return toGoSlice(i, t, output)
	}

	index := i * 32
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), index+32)
	}

	// Parse the given index output and check whether we need to read
	// a different offset and length based on the type (i.e. string, bytes)
	var returnOutput []byte
	switch t.Type.T {
	case abi.StringTy, abi.BytesTy: // variable arrays are written at the end of the return bytes
		// parse offset from which we should start reading
		offset := int(binary.BigEndian.Uint64(output[index+24 : index+32]))
		if offset+32 > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32)
		}
		// parse the size up until we should be reading
		size := int(binary.BigEndian.Uint64(output[offset+24 : offset+32]))
		if offset+32+size > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32+size)
		}

		// get the bytes for this return value
		returnOutput = output[offset+32 : offset+32+size]
	default:
		returnOutput = output[index : index+32]
	}

	// convert the bytes to whatever is specified by the ABI.
	switch t.Type.T {
	case abi.IntTy, abi.UintTy:
		return readInteger(t.Type.Kind, returnOutput), nil
	case abi.BoolTy:
		return readBool(returnOutput)
	case abi.AddressTy:
		return common.BytesToAddress(returnOutput), nil
	case abi.HashTy:
		return common.BytesToHash(returnOutput), nil
	case abi.BytesTy, abi.FixedBytesTy, abi.FunctionTy:
		return returnOutput, nil
	case abi.StringTy:
		return string(returnOutput), nil
	}
	return nil, fmt.Errorf("abi: unknown type %v", t.Type.T)
}

// toGoSliceType parses the input and casts it to the proper slice defined by the ABI
// argument in T.
func toGoSlice(i int, t abi.Argument, output []byte) (interface{}, error) {
	index := i * 32
	// The slice must, at very least be large enough for the index+32 which is exactly the size required
	// for the [offset in output, size of offset].
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go slice: insufficient size output %d require %d", len(output), index+32)
	}
	elem := t.Type.Elem

	// first we need to create a slice of the type
	var refSlice reflect.Value
	switch elem.T {
	case abi.IntTy, abi.UintTy, abi.BoolTy:
		// create a new reference slice matching the element type
		switch t.Type.Kind {
		case reflect.Bool:
			refSlice = reflect.ValueOf([]bool(nil))
		case reflect.Uint8:
			refSlice = reflect.ValueOf([]uint8(nil))
		case reflect.Uint16:
			refSlice = reflect.ValueOf([]uint16(nil))
		case reflect.Uint32:
			refSlice = reflect.ValueOf([]uint32(nil))
		case reflect.Uint64:
			refSlice = reflect.ValueOf([]uint64(nil))
		case reflect.Int8:
			refSlice = reflect.ValueOf([]int8(nil))
		case reflect.Int16:
			refSlice = reflect.ValueOf([]int16(nil))
		case reflect.Int32:
			refSlice = reflect.ValueOf([]int32(nil))
		case reflect.Int64:
			refSlice = reflect.ValueOf([]int64(nil))
		default:
			refSlice = reflect.ValueOf([]*big.Int(nil))
		}
	case abi.AddressTy: // address must be of slice Address
		refSlice = reflect.ValueOf([]common.Address(nil))
	case abi.HashTy: // hash must be of slice hash
		refSlice = reflect.ValueOf([]common.Hash(nil))
	case abi.FixedBytesTy:
		refSlice = reflect.ValueOf([][]byte(nil))
	default: // no other types are supported
		return nil, fmt.Errorf("abi: unsupported slice type %v", elem.T)
	}

	var slice []byte
	var size int
	var offset int
	if t.Type.IsSlice {
		// get the offset which determines the start of this array ...
		offset = int(binary.BigEndian.Uint64(output[index+24 : index+32]))
		if offset+32 > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go slice: offset %d would go over slice boundary (len=%d)", len(output), offset+32)
		}

		slice = output[offset:]
		// ... starting with the size of the array in elements ...
		size = int(binary.BigEndian.Uint64(slice[24:32]))
		slice = slice[32:]
		// ... and make sure that we've at the very least the amount of bytes
		// available in the buffer.
		if size*32 > len(slice) {
			return nil, fmt.Errorf("abi: cannot marshal in to go slice: insufficient size output %d require %d", len(output), offset+32+size*32)
		}

		// reslice to match the required size
		slice = slice[:size*32]
	} else if t.Type.IsArray {
		//get the number of elements in the array
		size = t.Type.SliceSize

		//check to make sure array size matches up
		if index+32*size > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)", len(output), index+32*size)
		}
		//slice is there for a fixed amount of times
		slice = output[index : index+size*32]
	}

	for i := 0; i < size; i++ {
		var (
			inter        interface{}             // interface type
			returnOutput = slice[i*32 : i*32+32] // the return output
			err          error
		)
		// set inter to the correct type (cast)
		switch elem.T {
		case abi.IntTy, abi.UintTy:
			inter = readInteger(t.Type.Kind, returnOutput)
		case abi.BoolTy:
			inter, err = readBool(returnOutput)
			if err != nil {
				return nil, err
			}
		case abi.AddressTy:
			inter = common.BytesToAddress(returnOutput)
		case abi.HashTy:
			inter = common.BytesToHash(returnOutput)
		case abi.FixedBytesTy:
			inter = returnOutput
		}
		// append the item to our reflect slice
		refSlice = reflect.Append(refSlice, reflect.ValueOf(inter))
	}

	return refSlice.Interface(), nil
}

func readInteger(kind reflect.Kind, b []byte) interface{} {
	switch kind {
	case reflect.Uint8:
		return uint8(b[len(b)-1])
	case reflect.Uint16:
		return binary.BigEndian.Uint16(b[len(b)-2:])
	case reflect.Uint32:
		return binary.BigEndian.Uint32(b[len(b)-4:])
	case reflect.Uint64:
		return binary.BigEndian.Uint64(b[len(b)-8:])
	case reflect.Int8:
		return int8(b[len(b)-1])
	case reflect.Int16:
		return int16(binary.BigEndian.Uint16(b[len(b)-2:]))
	case reflect.Int32:
		return int32(binary.BigEndian.Uint32(b[len(b)-4:]))
	case reflect.Int64:
		return int64(binary.BigEndian.Uint64(b[len(b)-8:]))
	default:
		return new(big.Int).SetBytes(b)
	}
}

func readBool(word []byte) (bool, error) {
	if len(word) != 32 {
		return false, fmt.Errorf("abi: fatal error: incorrect word length")
	}

	for i, b := range word {
		if b != 0 && i != 31 {
			return false, errBadBool
		}
	}

	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errBadBool
	}

}
