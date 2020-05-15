package liveview

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const (
	commonHeaderStartByte = 0xff
)

var payloadHeaderStartCode = []byte{0x24, 0x35, 0x68, 0x79}

var (
	wrongCommonHeaderStartBytesError  = errors.New("Expect start code to be equal 0xFF")
	wrongPayloadHeaderStartBytesError = errors.New("Payload start code different from {0x24, 0x35, 0x68, 0x79}")
)

type commonHeader struct {
	PayloadType    byte
	sequenceNumber uint16
	timeStamp      uint32
}

type payloadImageHeader struct {
	payloadDataSize uint32
	paddingSize     byte
	reserved        [120]byte
	JpegData        []byte
}
type LiveView struct {
	commonHeader  commonHeader
	payloadHeader payloadImageHeader
	reader        *bufio.Reader
	response      *http.Response
	url           string
}

func (liveview *LiveView) readCommonHeader() error {
	commonHeader := make([]byte, 1)
	_, err := liveview.reader.Read(commonHeader)
	if err != nil {
		return err
	}
	if commonHeader[0] != commonHeaderStartByte {
		return wrongCommonHeaderStartBytesError
	}

	payloadType := make([]byte, 1)
	_, err = liveview.reader.Read(payloadType)
	if err != nil {
		return err
	}
	liveview.commonHeader.PayloadType = payloadType[0]

	sequenceNumber := make([]byte, 2)
	_, err = liveview.reader.Read(sequenceNumber)
	if err != nil {
		return err
	}
	liveview.commonHeader.sequenceNumber = binary.BigEndian.Uint16(sequenceNumber)

	timeStamp := make([]byte, 4)
	_, err = liveview.reader.Read(timeStamp)
	if err != nil {
		return err
	}
	liveview.commonHeader.timeStamp = binary.BigEndian.Uint32(timeStamp)

	return nil
}

func (liveview *LiveView) readPayloadHeader() error {
	startCode := make([]byte, 4)
	_, err := liveview.reader.Read(startCode)
	if err != nil {
		return err
	}
	if !bytes.Equal(startCode, payloadHeaderStartCode) {
		return wrongPayloadHeaderStartBytesError
	}

	payloadDataSize := make([]byte, 3)
	_, err = liveview.reader.Read(payloadDataSize)
	if err != nil {
		return err
	}
	liveview.payloadHeader.payloadDataSize = binary.BigEndian.Uint32(append([]byte{0x00}, payloadDataSize...))

	paddingSize := make([]byte, 1)
	_, err = liveview.reader.Read(paddingSize)
	if err != nil {
		return err
	}
	liveview.payloadHeader.paddingSize = paddingSize[0]

	reservedBytes := make([]byte, 120)
	_, err = liveview.reader.Read(reservedBytes)
	if err != nil {
		return err
	}

	switch liveview.commonHeader.PayloadType {
	case 0x01:
		jpegDataVal := make([]byte, liveview.payloadHeader.payloadDataSize)
		_, err = liveview.reader.Read(jpegDataVal)
		if err != nil {
			return err
		}

		liveview.payloadHeader.JpegData = jpegDataVal

	case 0x02:
		frameInfo := make([]byte, 16)
		_, err = liveview.reader.Read(frameInfo)
		if err != nil {
			return err
		}
	}
	if liveview.payloadHeader.paddingSize != 0 {
		padding := make([]byte, liveview.payloadHeader.paddingSize)
		_, err = liveview.reader.Read(padding)
		if err != nil {
			return err
		}
	}

	return nil
}
func (liveview *LiveView) postReq(method string) error {
	_, err := http.Post(liveview.url+"/sony/camera", "application/json", bytes.NewBuffer([]byte(`{"method":`+method+`, "params": [], "id": 1, "version": "1.0"}`)))
	if err != nil {
		return err
	}

	return nil
}

func (liveview *LiveView) startRemoteControl() error {
	err := liveview.postReq("startRecMode")
	if err != nil {
		return err
	}
	return nil
}

func (liveview *LiveView) startLiveview() error {
	err := liveview.postReq("startLiveview")
	if err != nil {
		return err
	}

	log.Println("Start liveview")

	return nil
}

func (liveview *LiveView) stopLiveview() {
	err := liveview.postReq("stopLiveview")
	if err != nil {
		log.Printf("Failed to initialize sdl: %s\n", err)
		return
	}

	log.Println("Stop liveview")
}

func Start(liveviewURL string) (LiveView, error) {
	result := LiveView{
		commonHeader:  commonHeader{},
		payloadHeader: payloadImageHeader{},
		reader:        nil,
		response:      nil,
		url:           strings.TrimRight(liveviewURL, "/"),
	}

	err := result.startRemoteControl()
	if err != nil {
		return LiveView{}, err
	}

	err = result.startLiveview()
	if err != nil {
		return LiveView{}, err
	}

	return result, nil
}

func (liveview *LiveView) Stop() {
	liveview.stopLiveview()
	liveview.CloseResponse()
}

func (liveview *LiveView) CloseResponse() {
	liveview.response.Body.Close()
}

func (liveview *LiveView) Connect() error {
	response, err := http.Get(liveview.url + "/liveview/liveviewstream")
	if err != nil {
		return err
	}
	liveview.response = response
	liveview.reader = bufio.NewReader(liveview.response.Body)

	return nil
}

func (liveview *LiveView) FetchFrame() ([]byte, error) {
	err := liveview.readCommonHeader()
	if err != nil {
		return nil, fmt.Errorf("Failed to read common header: %s\n", err)
	}

	err = liveview.readPayloadHeader()
	if err != nil {
		return nil, fmt.Errorf("Failed to read payload header: %s\n", err)
	}

	return liveview.payloadHeader.JpegData, nil
}
