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
#include <sys/time.h>
*/
import "C"

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

type Safety int
type Position int
type Err int

type Jlog struct {
	ctx *C.jlog_ctx
}
type Id C.jlog_id

// Increment is used to increment the marker field in the C jlog_id struct.
func (id *Id) Increment() {
	id.marker++
}

// jlog_safety
const (
	JLOG_UNSAFE      Safety = C.JLOG_UNSAFE
	JLOG_ALMOST_SAFE Safety = C.JLOG_ALMOST_SAFE
	JLOG_SAFE        Safety = C.JLOG_SAFE
)

// jlog_position
const (
	JLOG_BEGIN Position = C.JLOG_BEGIN
	JLOG_END   Position = C.JLOG_END
)

// jlog_err
const (
	JLOG_ERR_SUCCESS            Err = C.JLOG_ERR_SUCCESS
	JLOG_ERR_ILLEGAL_INIT       Err = C.JLOG_ERR_ILLEGAL_INIT
	JLOG_ERR_ILLEGAL_OPEN       Err = C.JLOG_ERR_ILLEGAL_OPEN
	JLOG_ERR_OPEN               Err = C.JLOG_ERR_OPEN
	JLOG_ERR_NOTDIR             Err = C.JLOG_ERR_NOTDIR
	JLOG_ERR_CREATE_PATHLEN     Err = C.JLOG_ERR_CREATE_PATHLEN
	JLOG_ERR_CREATE_EXISTS      Err = C.JLOG_ERR_CREATE_EXISTS
	JLOG_ERR_CREATE_MKDIR       Err = C.JLOG_ERR_CREATE_MKDIR
	JLOG_ERR_CREATE_META        Err = C.JLOG_ERR_CREATE_META
	JLOG_ERR_LOCK               Err = C.JLOG_ERR_IDX_OPEN
	JLOG_ERR_IDX_OPEN           Err = C.JLOG_ERR_IDX_OPEN
	JLOG_ERR_IDX_SEEK           Err = C.JLOG_ERR_IDX_CORRUPT
	JLOG_ERR_IDX_CORRUPT        Err = C.JLOG_ERR_IDX_CORRUPT
	JLOG_ERR_IDX_WRITE          Err = C.JLOG_ERR_IDX_WRITE
	JLOG_ERR_IDX_READ           Err = C.JLOG_ERR_IDX_READ
	JLOG_ERR_FILE_OPEN          Err = C.JLOG_ERR_FILE_OPEN
	JLOG_ERR_FILE_SEEK          Err = C.JLOG_ERR_FILE_SEEK
	JLOG_ERR_FILE_CORRUPT       Err = C.JLOG_ERR_FILE_CORRUPT
	JLOG_ERR_FILE_READ          Err = C.JLOG_ERR_FILE_READ
	JLOG_ERR_FILE_WRITE         Err = C.JLOG_ERR_FILE_WRITE
	JLOG_ERR_META_OPEN          Err = C.JLOG_ERR_META_OPEN
	JLOG_ERR_ILLEGAL_WRITE      Err = C.JLOG_ERR_ILLEGAL_WRITE
	JLOG_ERR_ILLEGAL_CHECKPOINT Err = C.JLOG_ERR_ILLEGAL_CHECKPOINT
	JLOG_ERR_INVALID_SUBSCRIBER Err = C.JLOG_ERR_INVALID_SUBSCRIBER
	JLOG_ERR_ILLEGAL_LOGID      Err = C.JLOG_ERR_ILLEGAL_LOGID
	JLOG_ERR_SUBSCRIBER_EXISTS  Err = C.JLOG_ERR_SUBSCRIBER_EXISTS
	JLOG_ERR_CHECKPOINT         Err = C.JLOG_ERR_CHECKPOINT
	JLOG_ERR_NOT_SUPPORTED      Err = C.JLOG_ERR_NOT_SUPPORTED
)

func assertGTEZero(i C.int, function string, log Jlog) error {
	if int(i) < 0 {
		return fmt.Errorf("from %v, %v (%v)", function, log.ErrString(), log.Err())
	}
	return nil
}

func New(path string) Jlog {
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	return Jlog{(C.jlog_new(p))}
}

// XXX: jlog_set_error_func, setting with a C function unsupported.

func (log Jlog) RawSize() uint {
	return uint(C.jlog_raw_size(log.ctx))
}

func (log Jlog) Init() error {
	return assertGTEZero(C.jlog_ctx_init(log.ctx), "Init", log)
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

func (log Jlog) OpenWriter() error {
	return assertGTEZero(C.jlog_ctx_open_writer(log.ctx), "OpenWriter", log)
}

func (log Jlog) OpenReader(subscriber string) error {
	s := C.CString(subscriber)
	defer C.free(unsafe.Pointer(s))
	return assertGTEZero(C.jlog_ctx_open_reader(log.ctx, s), "OpenReader", log)
}

func (log Jlog) Close() {
	C.jlog_ctx_close(log.ctx)
}

func (log Jlog) AlterMode(mode int) {
	C.jlog_ctx_alter_mode(log.ctx, C.int(mode))
}

func (log Jlog) AlterJournalSize(size uint) error {
	return assertGTEZero(C.jlog_ctx_alter_journal_size(log.ctx, C.size_t(size)), "AlterJournalSize", log)
}

func (log Jlog) AlterSafety(safety Safety) error {
	return assertGTEZero(C.jlog_ctx_alter_safety(log.ctx, C.jlog_safety(safety)), "AlterSafety", log)
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

func (log Jlog) Write(message []byte) error {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&message))
	data := unsafe.Pointer(header.Data)
	return assertGTEZero(C.jlog_ctx_write(log.ctx, data, C.size_t(len(message))), "Write", log)
}

func (log Jlog) WriteMessage(message []byte, when time.Time) (int, error) {
	var tv C.struct_timeval
	duration := when.Sub(time.Now())
	tv.tv_sec = C.__time_t(duration.Seconds())
	tv.tv_usec = C.__suseconds_t(duration.Nanoseconds() / 1000)

	header := (*reflect.SliceHeader)(unsafe.Pointer(&message))
	data := unsafe.Pointer(header.Data)

	var msg C.jlog_message
	msg.mess_len = C.u_int32_t(len(message))
	msg.mess = data
	// The header fields are left uninitialized because they are not used
	// anywhere down the stracktrace of writing a message (only mess and mess_len
	// are used. Additionally, the header values are lower level metadata
	// information about timing and length. The length is already visible
	// in the []byte length, the timing of when a message is read is seems
	// less important.

	bytesWritten := C.jlog_ctx_write_message(log.ctx, &msg, &tv)

	return int(bytesWritten), assertGTEZero(bytesWritten, "WriteMessage", log)
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
