package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

const (
	defaultBlksize = int64(512)
)

type Server struct {
	remote   net.Addr
	conn     *net.UDPConn
	Buffer   *bytes.Buffer
	rootDir  string
	tftpFile *TftpFile
	notify   chan int
	timeout  time.Duration
	retries  int64
	blksize  int64
}

func NewServer(remote net.Addr, conn *net.UDPConn, buffer *bytes.Buffer, rootDir string, timeout time.Duration, retries int64) *Server {

	return &Server{
		remote:  remote,
		conn:    conn,
		Buffer:  buffer,
		notify:  make(chan int),
		timeout: timeout,
		retries: retries,
		rootDir: rootDir,
		blksize: defaultBlksize,
	}
}

func (h *Server) Serve() error {
	firstPacket, err := ioutil.ReadAll(h.Buffer)
	if err != nil {
		h.SendError(0, "Unable to read initial packet")
		return err
	}
	rrq, err := ParseRRQ(firstPacket)
	if err != nil {
		h.SendError(0, "This server only accepts reads")
		return err
	}

	h.tftpFile, err = NewTftpFile(h.rootDir, rrq.Filename, 512)
	if err != nil {
		h.SendError(1, "File not found")
		return err
	}
	defer h.tftpFile.Close()

	if len(rrq.Options) > 0 {
		oack_options := make(map[string]interface{})
		blksize, err := strconv.Atoi(rrq.Options["blksize"])
		if err == nil {
			oack_options["blksize"] = int64(blksize)
			h.blksize = int64(blksize)
		}

		_, has_tsize := rrq.Options["tsize"]

		if has_tsize {
			oack_options["tsize"] = h.tftpFile.Size()
		}

		oack := &OACK{
			Options: oack_options,
		}

		oackBytes, err := oack.Bytes()
		if err != nil {
			h.SendError(0, "Internal server error")
			return err
		}

		optionExchange := func() error {

			out, err := h.SendReceive(oackBytes)
			if err != nil {
				return err
			}

			ack, err := ParseACK(out)
			if err != nil {
				return err
			}
			if ack.Block_num != 0 {
				return fmt.Errorf("Ack is not 0: %d", ack.Block_num)
			}
			return nil
		}

		err = h.RetryLoop(optionExchange)
		if err != nil {
			h.SendError(0, "Error exchanging option")
			return nil
		}
	}

	h.tftpFile.SetBlockSize(h.blksize)
	blockNum := int64(1)

	for blockNum <= h.tftpFile.NumBlocks() {
		blockStep := func() error {
			b, err := h.getBlock(blockNum)
			if err != nil {
				return err
			}
			out, err := h.SendReceive(b)
			if err != nil {
				return err
			}
			ack, err := ParseACK(out)
			if err != nil {
				return err
			}
			if ack.Block_num != blockNum {
				return fmt.Errorf("block mismatch (exp: %d, recv: %d", blockNum, ack.Block_num)
			}
			return nil
		}

		err := h.RetryLoop(blockStep)

		if err == nil {
			blockNum++
		}
	}
	return nil
}

func (h *Server) getBlock(blockNum int64) ([]byte, error) {

	b, err := h.tftpFile.ReadBlock(blockNum)
	if err != nil && err != io.EOF {
		return nil, err
	}

	data := &DATA{
		Block_num: blockNum,
		Data:      b,
	}

	datab, err := data.Bytes()
	if err != nil {
		return nil, err
	}
	return datab, nil
}

func (h *Server) Notify() {
	h.notify <- 1
}

func (h *Server) SendReceive(input []byte) ([]byte, error) {

	h.conn.SetWriteDeadline(time.Now().Add(time.Second))
	n, err := h.conn.WriteTo(input, h.remote)
	if err != nil {
		return nil, fmt.Errorf("Failed to write: %v", err)
	}
	if n != len(input) {
		return nil, fmt.Errorf("Failed to write full array (%d < %d)", n, len(input))
	}

	timer := time.NewTimer(h.timeout)
	select {
	case <-timer.C:
		return nil, fmt.Errorf("Packet timed out")
	case <-h.notify:
		buffer, err := ioutil.ReadAll(h.Buffer)
		if err != nil {
			return nil, err
		}
		return buffer, nil
	}
}

func (h *Server) RetryLoop(f func() error) error {
	var err error
	for i := int64(0); i < h.retries; i++ {
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("Failed after %d retries [last error: %v]", h.retries, err)
}

func (h *Server) SendError(code int64, msg string) {
	errbytes, err2 := NewError(code, msg).Bytes()
	if err2 != nil {
		return
	}
	h.SendReceive(errbytes)
}
