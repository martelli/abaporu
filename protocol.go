package main

import (
	"bytes"
	"errors"
	"fmt"

	"encoding/binary"
)

func GetOpcode(raw []byte) (int64, error) {

	if len(raw) < 2 {
		return -1, errors.New("Less than 2 bytes available to parse opcode")
	}

	return int64(binary.BigEndian.Uint16(raw[:2])), nil
}

// RRQ
type RRQ struct {
	Opcode   int64
	Filename string
	Mode     string
	Options  map[string]string
}

func ParseRRQ(raw []byte) (*RRQ, error) {

	opcode, err := GetOpcode(raw)
	if err != nil {
		return nil, err
	}

	if opcode != 1 {
		return nil, errors.New("Cannot parse RRQ with opcode != 1")
	}

	filenameBuf, remain, err := ReadTillNull(raw[2:])
	if err != nil {
		return nil, err
	}
	if filenameBuf == nil {
		return nil, fmt.Errorf("Cannot find null terminator in for filename in RRQ (%v)", raw)
	}

	modeBuf, remain, err := ReadTillNull(remain)
	if err != nil {
		return nil, err
	}
	if filenameBuf == nil {
		return nil, fmt.Errorf("Cannot find null terminator in for filename in RRQ (%v)", raw)
	}

	var opt []byte
	var optvalue []byte
	options := make(map[string]string)

	for len(remain) > 0 {
		opt, remain, _ = ReadTillNull(remain)
		optvalue, remain, _ = ReadTillNull(remain)
		options[string(opt)] = string(optvalue)
	}

	return &RRQ{
		Opcode:   opcode,
		Filename: string(filenameBuf),
		Mode:     string(modeBuf),
		Options:  options,
	}, nil
}

// OACK
type OACK struct {
	Opcode  int64
	Options map[string]interface{}
}

func (oack *OACK) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, uint16(6))
	if err != nil {
		return nil, err
	}

	for k, v := range oack.Options {

		// key
		_, err := buf.WriteString(k)
		if err != nil {
			return nil, err
		}
		buf.WriteByte(0x00)

		// value
		switch vtyped := v.(type) {
		case string:
			_, err := buf.WriteString(vtyped)
			if err != nil {
				return nil, err
			}
		case int64:
			_, err := buf.WriteString(fmt.Sprintf("%s", vtyped))
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.New(fmt.Sprintf("Unrecognized type for %v", v))
		}
		buf.WriteByte(0x00)
	}

	return buf.Bytes(), nil
}

// DATA
type DATA struct {
	opcode    int64
	Block_num int64
	Data      []byte
}

func (data *DATA) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}

	err := binary.Write(buf, binary.BigEndian, uint16(3))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, uint16(data.Block_num))
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(data.Data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ACK
type ACK struct {
	opcode    int64
	Block_num int64
}

func ParseACK(raw []byte) (*ACK, error) {
	if len(raw) != 4 {
		return nil, fmt.Errorf("ACK frames need to be exactly 4 bytes long, got %d", len(raw))
	}
	return &ACK{
		Block_num: int64(binary.BigEndian.Uint16(raw[2:4])),
	}, nil
}

// ERROR
type ERROR struct {
	opcode        int64
	error_code    int64
	error_message string
}

func (error *ERROR) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}

	err := binary.Write(buf, binary.BigEndian, uint16(5))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, uint16(error.error_code))
	if err != nil {
		return nil, err
	}

	_, err = buf.WriteString(error.error_message)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(0x00)
	return buf.Bytes(), nil
}

func NewError(error_code int64, error_message string) *ERROR {
	return &ERROR{
		opcode:        4,
		error_code:    error_code,
		error_message: error_message,
	}
}
