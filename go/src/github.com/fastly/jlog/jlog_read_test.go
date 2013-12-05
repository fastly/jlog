package jlog_test

import (
	"github.com/fastly/jlog"
	"log"
	"testing"
)

func TestReading(t *testing.T) {
	initialize(t)

	writer, _ := jlog.NewWriter(pathname, nil)
	writer.AddSubscriber("reading", jlog.BEGIN)
	usageWritePayloads(pcnt, t) // write to the jlog

	reader, _ := jlog.NewReader(pathname, nil)
	e := reader.Open("reading")
	if e != nil {
		t.Errorf("Unable to open reader, error %v", reader.ErrString())
	}

	bytes, e := reader.Read()
	log.Printf("string: %v", string(bytes))
	for {
		bytesTemp, e := reader.Read()
		if bytesTemp == nil || e != nil {
			break
		}
	}

	usageWritePayloads(pcnt, t) // write to the jlog

	bytes2, e := reader.Read()
	log.Printf("string2: %v", string(bytes2))
	log.Printf("string: %v", string(bytes))
	log.Printf("if not equal, either the perl bindings have a problem or jlog has a problem")
}
