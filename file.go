package main

import (
	"fmt"
	"io"
	"os"
	"path"
)

type TftpFile struct {
	file      *os.File
	pos       int64
	blockSize int64
	numblocks int64
	fileSize  int64
}

func NewTftpFile(dir, filename string, blk int64) (*TftpFile, error) {

	full_path := path.Join(dir, filename)
	stat, err := os.Stat(full_path)

	if err != nil {
		return nil, err
	}

	if !stat.Mode().IsRegular() {
		return nil, err
	}

	file, err := os.Open(full_path)
	if err != nil {
		return nil, err
	}

	tf := &TftpFile{
		file:     file,
		fileSize: stat.Size(),
	}

	return tf, nil
}

func (tf *TftpFile) SetBlockSize(blk int64) {
	tf.numblocks = tf.fileSize/blk + 1
	tf.blockSize = blk
}

func (tf *TftpFile) NumBlocks() int64 {
	return tf.numblocks
}

func (tf *TftpFile) ReadBlock(n int64) ([]byte, error) {
	if n < 1 {
		return nil, fmt.Errorf("Counting starts at 1")
	}
	b := make([]byte, tf.blockSize)
	_, err := tf.file.Seek(tf.blockSize*(n-1), 0)
	if err != nil && err != io.EOF {
		tf.file.Close()
		return nil, err
	}
	rn, err := tf.file.Read(b)
	if err != nil {
		tf.file.Close()
		return nil, err
	}
	return b[:rn], nil
}

func (tf *TftpFile) Close() error {
	return tf.file.Close()
}

func (tf *TftpFile) Size() int64 {
	return tf.fileSize
}
