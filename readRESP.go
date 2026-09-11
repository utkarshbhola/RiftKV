package main

import (
	"bufio"
	"fmt"
	"strconv"
)

func readRESP(reader *bufio.Reader) (interface{}, error) {
	typeByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch typeByte {

	case '+':
		return readLine(reader)

	case '-':
		msg, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		return fmt.Errorf("%s", msg), nil

	case ':':
		msg, err := readLine(reader)
		if err != nil {
			return nil, err
		}

		number, err := strconv.Atoi(msg)
		if err != nil {
			return nil, err
		}

		return number, nil

	case '$':
		return readBulkString(reader)

	case '*':
		return readArray(reader)

	default:
		return nil, fmt.Errorf("unknown RESP type: %c", typeByte)
	}
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return line[:len(line)-2], nil
}

func readBulkString(reader *bufio.Reader) (string, error) {
	lengthLine, err := readLine(reader)
	if err != nil {
		return "", err
	}

	length, err := strconv.Atoi(lengthLine)
	if err != nil {
		return "", err
	}

	if length == -1 {
		return "", nil
	}

	data := make([]byte, length+2)

	_, err = reader.Read(data)
	if err != nil {
		return "", err
	}

	return string(data[:length]), nil
}

func readArray(reader *bufio.Reader) ([]interface{}, error) {
	lengthLine, err := readLine(reader)
	if err != nil {
		return nil, err
	}

	length, err := strconv.Atoi(lengthLine)
	if err != nil {
		return nil, err
	}

	if length == -1 {
		return nil, nil
	}

	result := make([]interface{}, length)

	for i := 0; i < length; i++ {
		value, err := readRESP(reader)
		if err != nil {
			return nil, err
		}

		result[i] = value
	}

	return result, nil
}
