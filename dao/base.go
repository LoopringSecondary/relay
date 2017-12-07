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

////////////////////////////////////////////////////
//
// base functions
//
////////////////////////////////////////////////////

// add single item
func (s *RdsServiceImpl) Add(item interface{}) error {
	return s.db.Create(item).Error
}

// del single item
func (s *RdsServiceImpl) Del(item interface{}) error {
	return s.db.Delete(item).Error
}

// select first item order by primary key asc
func (s *RdsServiceImpl) First(item interface{}) error {
	return s.db.First(item).Error
}

// select the last item order by primary key asc
func (s *RdsServiceImpl) Last(item interface{}) error {
	return s.db.Last(item).Error
}

// update single item
func (s *RdsServiceImpl) Save(item interface{}) error {
	return s.db.Save(item).Error
}

// find all items in table where primary key > 0
func (s *RdsServiceImpl) FindAll(item interface{}) error {
	return s.db.Table("lpr_orders").Find(item, s.db.Where("id > ", 0)).Error
}
