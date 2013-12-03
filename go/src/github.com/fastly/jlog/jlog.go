// A wrapper library for jlog.
//
// This wraps the C jlog.h library. jlog_set_error_func, jlog_ctx_write_message,
// and jlog_ctx_read_message are unimplemented because the C calls use either
// function passing or struct definitions that are not easily translatable
// from Go to C.
//
// Added documentation would be appreciated. Add it in jlog.h as well.
// This file uses LDFLAGS: -ljlog.
package jlog

/*
#cgo LDFLAGS: -ljlog
#include <stdlib.h>
#include <jlog.h>
*/
import "C"

import (
	"fmt"
	"reflect"
	"unsafe"
)

type Safety int
type Position int
type Err int

// jlog_safety
const (
	JLOG_UNSAFE Safety = iota
	JLOG_ALMOST_SAFE
	JLOG_SAFE
)

// jlog_position
const (
	JLOG_BEGIN Position = iota
	JLOG_END
)

// jlog_err
const (
	JLOG_ERR_SUCCESS Err = iota
	JLOG_ERR_ILLEGAL_INIT
	JLOG_ERR_ILLEGAL_OPEN
	JLOG_ERR_OPEN
	JLOG_ERR_NOTDIR
	JLOG_ERR_CREATE_PATHLEN
	JLOG_ERR_CREATE_EXISTS
	JLOG_ERR_CREATE_MKDIR
	JLOG_ERR_CREATE_META
	JLOG_ERR_LOCK
	JLOG_ERR_IDX_OPEN
	JLOG_ERR_IDX_SEEK
	JLOG_ERR_IDX_CORRUPT
	JLOG_ERR_IDX_WRITE
	JLOG_ERR_IDX_READ
	JLOG_ERR_FILE_OPEN
	JLOG_ERR_FILE_SEEK
	JLOG_ERR_FILE_CORRUPT
	JLOG_ERR_FILE_READ
	JLOG_ERR_FILE_WRITE
	JLOG_ERR_META_OPEN
	JLOG_ERR_ILLEGAL_WRITE
	JLOG_ERR_ILLEGAL_CHECKPOINT
	JLOG_ERR_INVALID_SUBSCRIBER
	JLOG_ERR_ILLEGAL_LOGID
	JLOG_ERR_SUBSCRIBER_EXISTS
	JLOG_ERR_CHECKPOINT
	JLOG_ERR_NOT_SUPPORTED
)

type Jlog struct {
	ctx *C.jlog_ctx
}
type Id *C.jlog_id

func New(path string) Jlog {
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	return Jlog{(C.jlog_new(p))}
}

// XXX: jlog_set_error_func, setting with a C function unsupported.

func (log Jlog) RawSize() uint {
	return uint(C.jlog_raw_size(log.ctx))
}

func (log Jlog) Init() int {
	return int(C.jlog_ctx_init(log.ctx))
}

func (log Jlog) GetCheckpoint(subscriber string, id Id) int {
	s := C.CString(subscriber)
	defer C.free(unsafe.Pointer(s))
	return int(C.jlog_get_checkpoint(log.ctx, s, id))
}

func (log Jlog) ListSubscribersDispose(subs []string) int {
	csubs := make([]*C.char, len(subs))
	for i, sub := range subs {
		csubs[i] = C.CString(sub)
	}
	defer func() {
		for i := range csubs {
			C.free(unsafe.Pointer(csubs[i]))
		}
	}()

	header := (*reflect.SliceHeader)(unsafe.Pointer(&csubs))
	data := unsafe.Pointer(header.Data)
	return int(C.jlog_ctx_list_subscribers_dispose(log.ctx, (**C.char)(data)))
}

func (log Jlog) ListSubscribers(subs [][]string) int {
	cssubs := make([][]*C.char, len(subs)) // c slices of subs
	for j, ssub := range subs {
		cssubs[j] = make([]*C.char, len(ssub))
		for i, sub := range ssub {
			cssubs[j][i] = C.CString(sub)
		}
	}
	defer func() {
		for _, cssub := range cssubs {
			for j := range cssub {
				C.free(unsafe.Pointer(cssub[j]))
			}
		}
	}()

	ccsubs := make([]**C.char, len(cssubs))
	for i, cssub := range cssubs {
		header := (*reflect.SliceHeader)(unsafe.Pointer(&cssub))
		data := unsafe.Pointer(header.Data)
		ccsubs[i] = (**C.char)(data)
	}

	header := (*reflect.SliceHeader)(unsafe.Pointer(&ccsubs))
	data := unsafe.Pointer(header.Data)
	return int(C.jlog_ctx_list_subscribers(log.ctx, (***C.char)(data)))
}

func (log Jlog) Err() int {
	return int(C.jlog_ctx_err(log.ctx))
}

func (log Jlog) ErrString() string {
	rChars := C.jlog_ctx_err_string(log.ctx)
	defer C.free(unsafe.Pointer(rChars))
	rStr := C.GoString(rChars)
	return rStr
}

func (log Jlog) Errno() int {
	return int(C.jlog_ctx_errno(log.ctx))
}

func (log Jlog) OpenWriter() int {
	return int(C.jlog_ctx_open_writer(log.ctx))
}

func (log Jlog) OpenReader(subscriber string) int {
	s := C.CString(subscriber)
	defer C.free(unsafe.Pointer(s))
	return int(C.jlog_ctx_open_reader(log.ctx, s))
}

func (log Jlog) Close() int {
	return int(C.jlog_ctx_close(log.ctx))
}

func (log Jlog) AlterMode(mode int) int {
	return int(C.jlog_ctx_alter_mode(log.ctx, C.int(mode)))
}

func (log Jlog) AlterJournalSize(size uint) int {
	return int(C.jlog_ctx_alter_journal_size(log.ctx, C.size_t(size)))
}

func (log Jlog) AlterSafety(safety Safety) int {
	return int(C.jlog_ctx_alter_safety(log.ctx, C.jlog_safety(safety)))
}

func (log Jlog) AddSubscriber(subscriber string, whence Position) int {
	c := C.CString(subscriber)
	defer C.free(unsafe.Pointer(c))
	return int(C.jlog_ctx_add_subscriber(log.ctx, c, C.jlog_position(whence)))
}

func (log Jlog) RemoveSubscriber(subscriber string) int {
	c := C.CString(subscriber)
	defer C.free(unsafe.Pointer(c))
	return int(C.jlog_ctx_remove_subscriber(log.ctx, c))
}

func (log Jlog) Write(message []byte) int {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&message))
	data := unsafe.Pointer(header.Data)
	return int(C.jlog_ctx_write(log.ctx, data, C.size_t(len(message))))
}

// XXX: jlog_ctx_write_message jlog_message unsupported

func (log Jlog) ReadInterval(firstMess, lastMess Id) int {
	return int(C.jlog_ctx_read_interval(log.ctx, firstMess, lastMess))
}

// XXX jlog_ctx_read_message, jlog_message unsupported

func (log Jlog) ReadCheckpoint(checkpoint Id) int {
	return int(C.jlog_ctx_read_checkpoint(log.ctx, checkpoint))
}

func (log Jlog) SnprintLogID(buffer []byte, checkpoint Id) int {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&buffer))
	data := unsafe.Pointer(header.Data)
	return int(C.jlog_snprint_logid((*C.char)(data), C.int(len(buffer)), checkpoint))
}

func (log Jlog) PendingReaders(ulog uint32) int {
	return int(C.__jlog_pending_readers(log.ctx, C.u_int32_t(ulog)))
}

func (log Jlog) FirstLogId(id Id) int {
	return int(C.jlog_ctx_first_log_id(log.ctx, id))
}

func (log Jlog) LastLogId(id Id) int {
	return int(C.jlog_ctx_last_log_id(log.ctx, id))
}

func (log Jlog) AdvanceId(current, start, finish Id) int {
	return int(C.jlog_ctx_advance_id(log.ctx, current, start, finish))
}
