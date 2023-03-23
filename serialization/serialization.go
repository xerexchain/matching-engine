package serialization

import (
	"bytes"
	"encoding/binary"
	"reflect"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/google/btree"
)

type Marshalable interface {
	Marshal(out *bytes.Buffer) error
}

type Unmarshalable interface {
	Unmarshal(in *bytes.Buffer) error
}

func UnmarshalUInt8(b *bytes.Buffer) (interface{}, error) {
	var res uint8
	err := binary.Read(b, binary.LittleEndian, &res)

	return res, err
}

func MarshalUInt8(in interface{}, out *bytes.Buffer) error {
	return binary.Write(out, binary.LittleEndian, in.(uint8))
}

func UnmarshalInt8(b *bytes.Buffer) (interface{}, error) {
	var res int8
	err := binary.Read(b, binary.LittleEndian, &res)

	return res, err
}

func MarshalInt8(in interface{}, out *bytes.Buffer) error {
	return binary.Write(out, binary.LittleEndian, in.(int8))
}

func UnmarshalUInt32(b *bytes.Buffer) (interface{}, error) {
	var res uint32
	err := binary.Read(b, binary.LittleEndian, &res)

	return res, err
}

func MarshalUInt32(in interface{}, out *bytes.Buffer) error {
	return binary.Write(out, binary.LittleEndian, in.(uint32))
}

func UnmarshalInt32(b *bytes.Buffer) (interface{}, error) {
	var res int32
	err := binary.Read(b, binary.LittleEndian, &res)

	return res, err
}

func MarshalInt32(in interface{}, out *bytes.Buffer) error {
	return binary.Write(out, binary.LittleEndian, in.(int32))
}

func UnmarshalInt64(b *bytes.Buffer) (interface{}, error) {
	var res int64
	err := binary.Read(b, binary.LittleEndian, &res)

	return res, err
}

func MarshalInt64(in interface{}, out *bytes.Buffer) error {
	return binary.Write(out, binary.LittleEndian, in.(int64))
}

func UnmarshalKeyVal(
	in *bytes.Buffer,
	f1 func(*bytes.Buffer) (interface{}, error), // key unmarshaler
	f2 func(*bytes.Buffer) (interface{}, error), // val unmarshaler
) (interface{}, interface{}, error) {
	if k, err := f1(in); err != nil {
		return nil, nil, err
	} else {
		if v, err := f2(in); err != nil {
			return nil, nil, err
		} else {
			return k, v, nil
		}
	}
}

func MarshalKeyVal(
	k interface{},
	v interface{},
	out *bytes.Buffer,
	f1 func(interface{}, *bytes.Buffer) error, // key marshaler
	f2 func(interface{}, *bytes.Buffer) error, // val marshaler
) error {
	if err := f1(k, out); err != nil {
		return err
	}

	if err := f2(v, out); err != nil {
		return err
	}

	return nil
}

func UnmarshalMap(
	b *bytes.Buffer,
	f1 func(*bytes.Buffer) (interface{}, error), // key unmarshaler
	f2 func(*bytes.Buffer) (interface{}, error), // val unmarshaler
) (map[interface{}]interface{}, error) {
	var val interface{}
	var err error

	if val, err = UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	map_ := make(map[interface{}]interface{}, size)

	for size > 0 {
		if k, v, err := UnmarshalKeyVal(b, f1, f2); err != nil {
			return nil, err
		} else {
			map_[k] = v
		}

		size--
	}

	return map_, nil
}

func MarshalMap(
	in interface{},
	out *bytes.Buffer,
	f1 func(interface{}, *bytes.Buffer) error, // key marshaler
	f2 func(interface{}, *bytes.Buffer) error, // val marshaler
) error {
	map_ := reflect.ValueOf(in)
	size := int32(map_.Len())

	if err := MarshalInt32(size, out); err != nil {
		return err
	}

	iter := map_.MapRange()

	for iter.Next() {
		k := iter.Key().Interface()
		v := iter.Value().Interface()

		if err := MarshalKeyVal(k, v, out, f1, f2); err != nil {
			return err
		}
	}

	return nil
}

func UnmarshalLinkedHashMap(
	b *bytes.Buffer,
	f1 func(*bytes.Buffer) (interface{}, error), // key unmarshaler
	f2 func(*bytes.Buffer) (interface{}, error), // val unmarshaler
) (*linkedhashmap.Map, error) {
	var val interface{}
	var err error

	if val, err = UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	linkedMap_ := linkedhashmap.New()

	for size > 0 {
		if k, v, err := UnmarshalKeyVal(b, f1, f2); err != nil {
			return nil, err
		} else {
			linkedMap_.Put(k, v)
		}

		size--
	}

	return linkedMap_, nil
}

func MarshalLinkedHashMap(
	linkedMap *linkedhashmap.Map,
	out *bytes.Buffer,
	f1 func(interface{}, *bytes.Buffer) error, // key marshaler
	f2 func(interface{}, *bytes.Buffer) error, // val marshaler
) error {
	size := int32(linkedMap.Size())

	if err := MarshalInt32(size, out); err != nil {
		return err
	}

	for _, k := range linkedMap.Keys() {
		v, _ := linkedMap.Get(k)

		if err := MarshalKeyVal(k, v, out, f1, f2); err != nil {
			return err
		}
	}

	return nil
}

func UnmarshalBtree(
	b *bytes.Buffer,
	f func(*bytes.Buffer) (interface{}, error), // val unmarshaler
) (*btree.BTree, error) {
	var val interface{}
	var err error

	if val, err = UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	btree_ := btree.New(4) // TODO param

	for size > 0 {
		if v, err := f(b); err != nil {
			return nil, err
		} else {
			btree_.ReplaceOrInsert(v.(btree.Item))
		}

		size--
	}

	return btree_, nil
}

func MarshalBtree(
	btree_ *btree.BTree,
	out *bytes.Buffer,
	f func(interface{}, *bytes.Buffer) error, // val marshaler
) error {
	size := int32(btree_.Len())

	if err := MarshalInt32(size, out); err != nil {
		return err
	}

	var err error

	btree_.Ascend(func(v btree.Item) bool {
		if err = f(v, out); err != nil {
			return false
		}

		return true
	})

	return err
}

func UnmarshalInt32Int64(b *bytes.Buffer) (map[int32]int64, error) {
	var val interface{}
	var err error

	if val, err = UnmarshalInt32(b); err != nil {
		return nil, err
	}

	size := val.(int32)
	res := make(map[int32]int64, size)

	for size > 0 {
		if k, v, err := UnmarshalKeyVal(
			b,
			UnmarshalInt32,
			UnmarshalInt64,
		); err != nil {
			return nil, err
		} else {
			res[k.(int32)] = v.(int64)
		}

		size--
	}

	return res, nil
}
