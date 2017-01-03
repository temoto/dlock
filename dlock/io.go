package dlock

import (
	"encoding/binary"
	"errors"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
)

var (
	Debug                = false
	ErrorMessageTooLarge = errors.New("Advertised message size exceeds configured limit")
)

func ReadMessage(r io.Reader, pb proto.Message, maxSize uint) error {
	var sizeBytes [4]byte
	_, err := r.Read(sizeBytes[:])
	if err != nil {
		return err
	}
	size := uint(binary.BigEndian.Uint32(sizeBytes[:]))
	if Debug {
		log.Printf("dlock.ReadMessage: expected size: %d", size)
	}

	if size > maxSize {
		return ErrorMessageTooLarge
	}
	buf := make([]byte, size)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	if Debug {
		log.Printf("dlock.ReadMessage: %x", buf)
	}

	err = proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}
	if Debug {
		log.Printf("dlock.ReadMessage: %v", pb)
	}

	return nil
}

func SendMessage(w io.Writer, pb proto.Message) error {
	buf, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	var sizeBytes [4]byte
	binary.BigEndian.PutUint32(sizeBytes[:], uint32(len(buf)))
	if _, err = w.Write(sizeBytes[:]); err != nil {
		return err
	}
	if _, err = w.Write(buf); err != nil {
		return err
	}
	return nil
}
