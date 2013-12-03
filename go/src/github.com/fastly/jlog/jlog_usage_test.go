package jlog_test

import (
	"github.com/fastly/jlog"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

var payload string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var pcnt int = 1000000
var pathname string

func initialize(t *testing.T) {
	f, e := ioutil.TempFile("/tmp", "gojlogtest.")
	if e != nil {
		t.Errorf("unable to create tempfile")
	}
	pathname = f.Name()
	log.Println(pathname)
	f.Close()
	e = os.Remove(pathname)
	if e != nil {
		t.Errorf("unable to remove tempfile")
	}
	ctx := jlog.New(pathname)
	log.Println(ctx)
	ctx.AlterSafety(jlog.JLOG_SAFE)
	ctx.Init()
	ctx.Close()
}

func testSubscriber(subscriber string, t *testing.T) {
	log.Println("adding subscriber", subscriber, "to pathname", pathname)
	ctx := jlog.New(pathname)
	e := ctx.AddSubscriber(subscriber, jlog.JLOG_BEGIN)
	if e != nil && ctx.Err() != jlog.JLOG_ERR_SUBSCRIBER_EXISTS {
		t.Errorf("test subscriber, error %s != subscriber exists",
			ctx.ErrString())
	}
}

func assertSubscriber(subscriber string, expectingIt bool, t *testing.T) {
	log.Println("checking subscriber", subscriber)
	ctx := jlog.New(pathname)
	subs, e := ctx.ListSubscribers()
	if e != nil {
		t.Errorf("assert subscriber, error %s", ctx.ErrString())
	}
	for _, v := range subs {
		if subscriber == v {
			if expectingIt {
				return
			} else {
				t.Errorf("found matching subcriber %v but not expecting it",
					subscriber)
			}
		}
	}
	if expectingIt {
		t.Errorf("Unable to find the expected subscriber %v", subscriber)
	}
}

func writePayloads(cnt int, t *testing.T) {
	ctx := jlog.New(pathname)
	e := ctx.OpenWriter()
	if e != nil {
		t.Errorf("Unable to open writer, error %v", ctx.ErrString())
	}
	log.Printf("writing out %d %d byte payloads", cnt, len(payload))
	bytePayload := []byte(payload)
	for i := 0; i < cnt; i++ {
		ctx.Write(bytePayload)
	}
}

func readCheck(subscriber string, expect int, sizeup bool, t *testing.T) {
	cnt := 0
	ctx := jlog.New(pathname)
	e := ctx.OpenReader(subscriber)
	if e != nil {
		t.Errorf("Unable to open reader, error %v", ctx.ErrString())
	}
	start := ctx.RawSize()
	for {
		var startID, finishID jlog.Id
		count, e := ctx.ReadInterval(&startID, &finishID)
		if e != nil {
			t.Errorf("Unable to read interval, error %v", ctx.ErrString())
		}
		batchsize := count
		// Java copies jlog.Id's into each other. It also uses object semantics which
		// makes that POINTLESS so whoever wrote the interval whatever semantics
		// in that test doesn't know what they're doing. cur and chkpt point to the
		// same thing, the startID.
		if batchsize == 0 {
			break
		}
		for i := 0; i < batchsize; i++ {
			_, e := ctx.ReadMessage(&startID)
			if i != 0 {
				startID.Increment()
			}
			if e != nil {
				t.Errorf("Unable to read message, error %v", ctx.ErrString())
			}
			cnt++
		}
		e = ctx.ReadCheckpoint(&startID)
		if e != nil {
			t.Errorf("Unable to read checkpoint, error %v", ctx.ErrString())
		}
	}
	if cnt != expect {
		t.Errorf("got wrong read count: %v != expect %v", cnt, expect)
	}
	end := ctx.RawSize()
	if sizeup {
		log.Printf("checking that size increased")
	} else {
		log.Printf("checking that size decreased")
	}
	if sizeup && end < start {
		t.Errorf("size didn't increase as expected")
	}
	if !sizeup && end > start {
		t.Errorf("size didn't decrease as expected")
	}
}

// A test ripped from the java test file.
func TestUsage(t *testing.T) {
	initialize(t)
	testSubscriber("testing", t)
	assertSubscriber("testing", true, t)
	testSubscriber("witness", t)
	assertSubscriber("witness", true, t)
	assertSubscriber("badguy", false, t)
	writePayloads(pcnt, t)
	readCheck("witness", pcnt, true, t)
	readCheck("testing", pcnt, false, t)
}
