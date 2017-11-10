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

package dao

// order amountS 上限1e30

type Order struct {
	ID                    int     `gorm:"column:id;primary_key;"`
	Owner                 string  `gorm:"column:owner;type:varchar(42)"`
	OrderHash             string  `gorm:"column:order_hash;type:varchar(82);unique_index"`
	TokenS                string  `gorm:"column:token_s;type:varchar(42)"`
	TokenB                string  `gorm:"column:token_b;type:varchar(42)"`
	AmountS               []byte  `gorm:"column:amount_s;type:varchar(30)"`
	AmountB               []byte  `gorm:"column:amount_b;type:varchar(30)"`
	CreateTime            int     `gorm:"column:create_time"`
	Ttl                   int     `gorm:"column:ttl"`
	Salt                  int     `gorm:"column:salt"`
	LrcFee                []byte  `gorm:"column:lrc_fee;type:varchar(128)"`
	BuyNoMoreThanAmountB  bool    `gorm:"column:buy_nomore_than_amountb;type:bit"`
	MarginSplitPercentage uint8   `gorm:"column:margin_split_percentage;type:tinyint(4)"`
	V                     uint8   `gorm:"column:v;type:tinyint(4)"`
	R                     string  `gorm:"column:r;type:varchar(62)"`
	S                     string  `gorm:"column:s;type:varchar(62)"`
	Price                 float64 `gorm:"column:price;type:decimal(16,8);"`
}
