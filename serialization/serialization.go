package serialization

import (
	"bytes"
	"encoding/binary"
)

func UnmarshalInt32Int64(in *bytes.Buffer) (interface{}, error) {
	var size int32

	if err := binary.Read(in, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	res := make(map[int32]int64, size)

	var k int32
	var v int64

	for size > 0 {
		if err := binary.Read(in, binary.LittleEndian, &k); err != nil {
			return nil, err
		}

		if err := binary.Read(in, binary.LittleEndian, &v); err != nil {
			return nil, err
		}

		res[k] = v
		size--
	}

	return res, nil
}

// TODO Ref: duplicate code
func UnmarshalInt32Interface(in *bytes.Buffer, f func(*bytes.Buffer) (interface{}, error)) (interface{}, error) {
	var size int32

	if err := binary.Read(in, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	res := make(map[int32]interface{}, size)

	var k int32

	for size > 0 {
		if err := binary.Read(in, binary.LittleEndian, &k); err != nil {
			return nil, err
		}

		v, err := f(in)

		if err != nil {
			return nil, err
		}

		res[k] = v
		size--
	}

	return res, nil
}

func UnmarshalInt64Interface(in *bytes.Buffer, f func(*bytes.Buffer) (interface{}, error)) (interface{}, error) {
	var size int32

	if err := binary.Read(in, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	res := make(map[int64]interface{}, size)

	var k int64

	for size > 0 {
		if err := binary.Read(in, binary.LittleEndian, &k); err != nil {
			return nil, err
		}

		v, err := f(in)

		if err != nil {
			return nil, err
		}

		res[k] = v
		size--
	}

	return res, nil
}

func MarshalInt32Int64(in interface{}, out *bytes.Buffer) error {
	m := in.(map[int32]int64)
	numItems := int32(len(m))

	if err := binary.Write(out, binary.LittleEndian, numItems); err != nil {
		return err
	}

	for k, v := range m {
		if err := binary.Write(out, binary.LittleEndian, k); err != nil {
			return err
		}

		if err := binary.Write(out, binary.LittleEndian, v); err != nil {
			return err
		}
	}

	return nil
}

// TODO Ref: duplicate code
func MarshalInt32Interface(in interface{}, out *bytes.Buffer, f func(interface{}, *bytes.Buffer) error) error {
	m := in.(map[int32]interface{})
	numItems := int32(len(m))

	if err := binary.Write(out, binary.LittleEndian, numItems); err != nil {
		return err
	}

	for k, v := range m {
		if err := binary.Write(out, binary.LittleEndian, k); err != nil {
			return err
		}

		if err := f(v, out); err != nil {
			return err
		}
	}

	return nil
}

func MarshalInt64Interface(in interface{}, out *bytes.Buffer, f func(interface{}, *bytes.Buffer) error) error {
	m := in.(map[int64]interface{})
	numItems := int32(len(m))

	if err := binary.Write(out, binary.LittleEndian, numItems); err != nil {
		return err
	}

	for k, v := range m {
		if err := binary.Write(out, binary.LittleEndian, k); err != nil {
			return err
		}

		if err := f(v, out); err != nil {
			return err
		}
	}

	return nil
}
