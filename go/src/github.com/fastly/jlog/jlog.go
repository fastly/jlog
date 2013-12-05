// A wrapper library for jlog.
//
// This wraps the C jlog.h library. jlog_set_error_func is unimplemented because
// defining and passing a C function around is not easy to do in Go.
//
// Added documentation would be appreciated. Add it in jlog.h as well.
// This file uses LDFLAGS: -ljlog.
package jlog

/*
#cgo LDFLAGS: -ljlog
#include <jlog.h>
#include <stdlib.h>
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

type Jlog struct {
	ctx *C.jlog_ctx
}

// Options to use when creating a new Reader or Writer.
type Options struct {
	CreateSafety    Safety
	JournalSize     uint
	ExclusiveNew    bool // Fail if the file exists
	FilePermissions int  // an octal value of the file permissions
}

type Id C.jlog_id

// Increment is used to increment the marker field in the C jlog_id struct.
func (id *Id) Increment() {
	id.marker++
}

// jlog_safety
const (
	UNSAFE      Safety = C.JLOG_UNSAFE
	ALMOST_SAFE Safety = C.JLOG_ALMOST_SAFE
	SAFE        Safety = C.JLOG_SAFE
)

// jlog_position
const (
	BEGIN Position = C.JLOG_BEGIN
	END   Position = C.JLOG_END
)

// jlog_err
const (
	ERR_SUCCESS            Err = C.JLOG_ERR_SUCCESS
	ERR_ILLEGAL_INIT       Err = C.JLOG_ERR_ILLEGAL_INIT
	ERR_ILLEGAL_OPEN       Err = C.JLOG_ERR_ILLEGAL_OPEN
	ERR_OPEN               Err = C.JLOG_ERR_OPEN
	ERR_NOTDIR             Err = C.JLOG_ERR_NOTDIR
	ERR_CREATE_PATHLEN     Err = C.JLOG_ERR_CREATE_PATHLEN
	ERR_CREATE_EXISTS      Err = C.JLOG_ERR_CREATE_EXISTS
	ERR_CREATE_MKDIR       Err = C.JLOG_ERR_CREATE_MKDIR
	ERR_CREATE_META        Err = C.JLOG_ERR_CREATE_META
	ERR_LOCK               Err = C.JLOG_ERR_IDX_OPEN
	ERR_IDX_OPEN           Err = C.JLOG_ERR_IDX_OPEN
	ERR_IDX_SEEK           Err = C.JLOG_ERR_IDX_CORRUPT
	ERR_IDX_CORRUPT        Err = C.JLOG_ERR_IDX_CORRUPT
	ERR_IDX_WRITE          Err = C.JLOG_ERR_IDX_WRITE
	ERR_IDX_READ           Err = C.JLOG_ERR_IDX_READ
	ERR_FILE_OPEN          Err = C.JLOG_ERR_FILE_OPEN
	ERR_FILE_SEEK          Err = C.JLOG_ERR_FILE_SEEK
	ERR_FILE_CORRUPT       Err = C.JLOG_ERR_FILE_CORRUPT
	ERR_FILE_READ          Err = C.JLOG_ERR_FILE_READ
	ERR_FILE_WRITE         Err = C.JLOG_ERR_FILE_WRITE
	ERR_META_OPEN          Err = C.JLOG_ERR_META_OPEN
	ERR_ILLEGAL_WRITE      Err = C.JLOG_ERR_ILLEGAL_WRITE
	ERR_ILLEGAL_CHECKPOINT Err = C.JLOG_ERR_ILLEGAL_CHECKPOINT
	ERR_INVALID_SUBSCRIBER Err = C.JLOG_ERR_INVALID_SUBSCRIBER
	ERR_ILLEGAL_LOGID      Err = C.JLOG_ERR_ILLEGAL_LOGID
	ERR_SUBSCRIBER_EXISTS  Err = C.JLOG_ERR_SUBSCRIBER_EXISTS
	ERR_CHECKPOINT         Err = C.JLOG_ERR_CHECKPOINT
	ERR_NOT_SUPPORTED      Err = C.JLOG_ERR_NOT_SUPPORTED
)

func assertGTEZero(i C.int, function string, log Jlog) error {
	if int(i) < 0 {
		return fmt.Errorf("from %v, %v (%v)", function, log.ErrString(), log.Err())
	}
	return nil
}

func newJlog(path string, o *Options) (Jlog, error) {
	var e error

	options := Options{
		CreateSafety:    SAFE,
		JournalSize:     1024 * 1024,
		ExclusiveNew:    false,
		FilePermissions: 0640,
	}
	if o != nil {
		options = *o
	}

	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))

	log := Jlog{ctx: C.jlog_new(p)}
	// Setup based on options.
	e = assertGTEZero(C.jlog_ctx_alter_journal_size(log.ctx,
		C.size_t(options.JournalSize)), "New, alter journal size", log)
	if e != nil {
		return log, e
	}
	e = assertGTEZero(C.jlog_ctx_alter_mode(log.ctx,
		C.int(options.FilePermissions)), "New, alter mode", log)
	if e != nil {
		return log, e
	}
	e = assertGTEZero(C.jlog_ctx_alter_safety(log.ctx,
		C.jlog_safety(options.CreateSafety)), "New, alter safety", log)
	if e != nil {
		return log, e
	}
	e = assertGTEZero(C.jlog_ctx_init(log.ctx), "New, init", log)
	if e != nil && (log.Err() != ERR_CREATE_EXISTS || options.ExclusiveNew == true) {
		return log, e
	}
	log.Close()
	log = Jlog{ctx: C.jlog_new(p), Path: path}
	return log, nil // e could be set from ERR_CREATE_EXISTS
}

// XXX: jlog_set_error_func, setting with a C function unsupported.

func (log Jlog) RawSize() uint {
	return uint(C.jlog_raw_size(log.ctx))
}

func (log Jlog) ListSubscribers() ([]string, error) {
	var csubs **C.char
	r := int(C.jlog_ctx_list_subscribers(log.ctx, &csubs))
	if r < 0 {
		return nil, assertGTEZero(C.int(r), "ListSubscribers", log)
	}

	subs := make([]string, r)
	chrptrsz := unsafe.Sizeof(csubs) // sizeof char *
	base := uintptr(unsafe.Pointer(csubs))
	for i := uintptr(0); i < uintptr(r); i++ {
		curptr := *(**C.char)(unsafe.Pointer(base + i*chrptrsz))
		subs[i] = C.GoString(curptr)
	}
	C.jlog_ctx_list_subscribers_dispose(log.ctx, csubs)
	return subs, nil
}

// Err returns the last error (an enum).
func (log Jlog) Err() Err {
	return Err(C.jlog_ctx_err(log.ctx))
}

// ErrString returns the string representation of the last error.
func (log Jlog) ErrString() string {
	rChars := C.jlog_ctx_err_string(log.ctx)
	// no free because these are static char *
	rStr := C.GoString(rChars)
	return rStr
}

// Errno returns the last errno.
func (log Jlog) Errno() int {
	return int(C.jlog_ctx_errno(log.ctx))
}

func (log Jlog) OpenReader(subscriber string) error {
	s := C.CString(subscriber)
	defer C.free(unsafe.Pointer(s))
	return assertGTEZero(C.jlog_ctx_open_reader(log.ctx, s), "OpenReader", log)
}

func (log Jlog) Close() {
	C.jlog_ctx_close(log.ctx)
}

func (log Jlog) AddSubscriber(subscriber string, whence Position) error {
	c := C.CString(subscriber)
	defer C.free(unsafe.Pointer(c))
	return assertGTEZero(C.jlog_ctx_add_subscriber(log.ctx, c, C.jlog_position(whence)), "AddSubscriber", log)
}

func (log Jlog) RemoveSubscriber(subscriber string) error {
	c := C.CString(subscriber)
	defer C.free(unsafe.Pointer(c))
	return assertGTEZero(C.jlog_ctx_remove_subscriber(log.ctx, c), "RemoveSubscriber", log)
}

// ReadInterval changes the underlying Id's as necessary.
func (log Jlog) ReadInterval(firstMess, lastMess *Id) (int, error) {
	fid := C.jlog_id(*firstMess)
	lid := C.jlog_id(*lastMess)
	count := C.jlog_ctx_read_interval(log.ctx, &fid, &lid)
	e := assertGTEZero(count, "ReadInterval", log)
	*firstMess = Id(fid)
	*lastMess = Id(lid)
	return int(count), e
}

// ReadMessage changes the underlying Id's as necessary.
func (log Jlog) ReadMessage(id *Id) ([]byte, error) {
	cid := C.jlog_id(*id)
	var m C.jlog_message
	e := assertGTEZero(C.jlog_ctx_read_message(log.ctx, &cid, &m), "ReadMessage", log)
	var s []byte
	header := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	header.Data = uintptr(m.mess)
	header.Len = int(m.mess_len)
	header.Cap = int(m.mess_len)
	*id = Id(cid)
	return s, e
}

// ReadCheckpoint changes the underlying Id as necessary.
func (log Jlog) ReadCheckpoint(checkpoint *Id) error {
	cid := C.jlog_id(*checkpoint)
	e := assertGTEZero(C.jlog_ctx_read_checkpoint(log.ctx, &cid), "ReadCheckpoint", log)
	*checkpoint = Id(cid)
	return e
}

// TODO change to an inspect call that returns a string containing info on all ids
// SnprintLogId does not change the underlying Id.
func (log Jlog) SnprintLogId(buffer []byte, checkpoint *Id) (int, error) {
	cid := C.jlog_id(*checkpoint)
	header := (*reflect.SliceHeader)(unsafe.Pointer(&buffer))
	data := unsafe.Pointer(header.Data)
	bWritten := C.jlog_snprint_logid((*C.char)(data), C.int(len(buffer)), &cid)
	e := assertGTEZero(bWritten, "SnprintLogId", log)
	*checkpoint = Id(cid)
	return int(bWritten), e
}

func (log Jlog) PendingReaders(ulog uint32) (int, error) {
	readers := C.__jlog_pending_readers(log.ctx, C.u_int32_t(ulog))
	e := assertGTEZero(readers, "PendingReaders", log)
	return int(readers), e
}

// FirstLogId changes the underlying Id as necessary.
func (log Jlog) FirstLogId(id *Id) error {
	cid := C.jlog_id(*id)
	e := assertGTEZero(C.jlog_ctx_first_log_id(log.ctx, &cid), "FirstLogId", log)
	*id = Id(cid)
	return e
}

// LastLogId changes the underlying Id as necessary.
func (log Jlog) LastLogId(id *Id) error {
	cid := C.jlog_id(*id)
	e := assertGTEZero(C.jlog_ctx_last_log_id(log.ctx, &cid), "LastLogId", log)
	*id = Id(cid)
	return e
}

// AdvanceId changes the underlying Ids as necessary.
func (log Jlog) AdvanceId(current, start, finish *Id) error {
	cid := C.jlog_id(*current)
	sid := C.jlog_id(*start)
	fid := C.jlog_id(*finish)
	e := assertGTEZero(C.jlog_ctx_advance_id(log.ctx, &cid, &sid, &fid), "AdvanceId", log)
	*current = Id(cid)
	*start = Id(sid)
	*finish = Id(fid)
	return e
}
