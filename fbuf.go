package irtt

import "fmt"

// field
type field struct {
	pos int
	len int
	cap int
}

// field index
type fidx int

// fbuf provides access to fields in a byte buffer, each with a position, length
// and capacity. Each field must have a length of either 0 or the field's
// capacity, so that the structure of the buffer can be externalized simply as
// which fields are set. tlen sets a target buffer length, and the payload is
// the padding after the fields needed to meet the target length. The length of
// the buffer must always be at least the length of the fields.
type fbuf struct {
	// buffer
	buf []byte

	// fields
	fields []field

	// target length
	tlen int
}

func newFbuf(fields []field, tlen int, cap int) *fbuf {
	blen, fcap := sumFields(fields)
	if tlen > blen {
		blen = tlen
	}
	if fcap > cap {
		cap = fcap
	}
	return &fbuf{make([]byte, blen, cap), fields, tlen}
}

func (fb *fbuf) validate() error {
	flen, fcap := fb.sumFields()
	if flen > len(fb.buf) {
		return Errorf(FieldsLengthTooLarge,
			"fields length exceeds buffer length, %d > %d", flen, len(fb.buf))
	}
	if fcap > cap(fb.buf) {
		return Errorf(FieldsCapacityTooLarge,
			"fields capacity exceeds buffer capacity, %d > %d", fcap, cap(fb.buf))
	}
	return nil
}

// setFields and addFields are used when changing fields for an existing buffer

func (fb *fbuf) setFields(fidxs []fidx, setLen bool) error {
	pos := 0
	j := 0
	for i := 0; i < len(fidxs); i, j = i+1, j+1 {
		for ; j < len(fb.fields); j++ {
			if j == int(fidxs[i]) {
				fb.fields[j].pos = pos
				fb.fields[j].len = fb.fields[j].cap
				pos += fb.fields[j].len
				break
			}
			fb.fields[j].len = 0
			fb.fields[j].pos = pos
		}
	}
	for ; j < len(fb.fields); j++ {
		fb.fields[j].len = 0
		fb.fields[j].pos = pos
	}
	if setLen {
		fb.setLen(pos)
	}
	return fb.validate()
}

func (fb *fbuf) addFields(fidxs []fidx, setLen bool) error {
	pos := 0
	j := 0
	for i := 0; i < len(fidxs); i, j = i+1, j+1 {
		for ; j < len(fb.fields); j++ {
			if j == int(fidxs[i]) {
				fb.fields[j].pos = pos
				fb.fields[j].len = fb.fields[j].cap
				pos += fb.fields[j].len
				break
			}
			fb.fields[j].pos = pos
			pos += fb.fields[j].len
		}
	}
	for ; j < len(fb.fields); j++ {
		fb.fields[j].pos = pos
		pos += fb.fields[j].len
	}
	if setLen {
		fb.setLen(pos)
	}
	return fb.validate()
}

// setters

func (fb *fbuf) set(f fidx, b []byte) {
	p, l, c := fb.field(f)
	if len(b) != c {
		panic(fmt.Sprintf("set for field %d with size %d != field cap %d", f, len(b), c))
	}
	if l != c {
		fb.setFieldLen(f, c)
	}
	copy(fb.buf[p:p+c], b)
}

func (fb *fbuf) setTo(f fidx) []byte {
	p, l, c := fb.field(f)
	if l != c {
		fb.setFieldLen(f, c)
	}
	return fb.buf[p : p+c]
}

func (fb *fbuf) setb(f fidx, b byte) {
	p, l, c := fb.field(f)
	if c != 1 {
		panic("setb only for one byte fields")
	}
	if l != 1 {
		fb.setFieldLen(f, 1)
	}
	fb.buf[p] = b
}

func (fb *fbuf) setPayload(b []byte) {
	flen := fb.sumLens()
	fb.buf = fb.buf[:flen+len(b)]
	copy(fb.buf[flen:], b)
}

func (fb *fbuf) zeroPayload() {
	zero(fb.payload())
}

func (fb *fbuf) zero(f fidx) {
	zero(fb.setTo(f))
}

// getters

func (fb *fbuf) get(f fidx) []byte {
	p, l, _ := fb.field(f)
	return fb.buf[p : p+l]
}

func (fb *fbuf) getb(f fidx) byte {
	p, l, _ := fb.field(f)
	if l != 1 {
		panic(fmt.Sprintf("getb for non-byte field %d", f))
	}
	return fb.buf[p]
}

func (fb *fbuf) isset(f fidx) bool {
	return fb.fields[f].len > 0
}

func (fb *fbuf) bytes() []byte {
	return fb.buf
}

func (fb *fbuf) payload() []byte {
	flen := fb.sumLens()
	return fb.buf[flen:]
}

// length and capacity

func (fb *fbuf) length() int {
	return len(fb.buf)
}

// func (fb *fbuf) capacity() int {
// 	return cap(fb.buf)
// }

func (fb *fbuf) setLen(tlen int) int {
	fb.tlen = tlen
	flen := fb.sumLens()
	l := tlen
	if l < flen {
		l = flen
	}
	if l > cap(fb.buf) {
		l = cap(fb.buf)
	}
	fb.buf = fb.buf[:l]
	return l
}

// removal

func (fb *fbuf) remove(f fidx) {
	if fb.fields[f].len > 0 {
		fb.setFieldLen(f, 0)
	}
}

// internal methods

func (fb *fbuf) field(f fidx) (int, int, int) {
	return fb.fields[f].pos, fb.fields[f].len, fb.fields[f].cap
}

func (fb *fbuf) setFieldLen(f fidx, newlen int) {
	p, l, _ := fb.field(f)
	grow := newlen - l
	if grow != 0 {
		// grow or shrink the buffer and shift bytes
		//fmt.Printf("f=%d, newlen=%d, l=%d, len=%d, cap=%d, grow=%d\n",
		//	f, newlen, l, len(fb.buf), cap(fb.buf), grow)
		fb.buf = fb.buf[:len(fb.buf)+grow]
		copy(fb.buf[p+grow:], fb.buf[p:])

		// update field length
		fb.fields[f].len = newlen

		// update field positions
		newp := fb.fields[f].pos
		for i := f; i < fidx(len(fb.fields)); i++ {
			fb.fields[i].pos = newp
			newp += fb.fields[i].len
		}

		// update total field length and reset to target length
		flen := fb.sumLens()
		if fb.tlen >= flen {
			fb.buf = fb.buf[0:fb.tlen]
		}
	}
}

func (fb *fbuf) sumFields() (flen int, fcap int) {
	return sumFields(fb.fields)
}

func (fb *fbuf) sumLens() (flen int) {
	for _, f := range fb.fields {
		flen += f.len
	}
	return
}

func sumFields(fields []field) (flen int, fcap int) {
	for _, f := range fields {
		flen += f.len
		fcap += f.cap
	}
	return
}
