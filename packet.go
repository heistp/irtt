package irtt

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/binary"
	"hash"
	"io"
	"time"
)

// -------------------------------------------------------------------------------
// | Oct |        0        |        1        |        2        |        3        |
// -------------------------------------------------------------------------------
// |     | 0 1 2 3 4 5 6 7 | 0 1 2 3 4 5 6 7 | 0 1 2 3 4 5 6 7 | 0 1 2 3 4 5 6 7 |
// |------------------------------------------------------------------------------
// |  0  |                        Magic                        |      Flags      |
// |------------------------------------------------------------------------------
// |  4  |                              Conn Token                               |
// |------------------------------------------------------------------------------
// |  8  |                              Conn Token                               |
// |------------------------------------------------------------------------------
// | 12  |                                 Seqno                                 |
// |------------------------------------------------------------------------------
// | 16  |                        HMAC (if HMAC flag set)                        |
// |------------------------------------------------------------------------------
// | 20  |                        HMAC (if HMAC flag set)                        |
// |------------------------------------------------------------------------------
// | 24  |                        HMAC (if HMAC flag set)                        |
// |------------------------------------------------------------------------------
// | 28  |                        HMAC (if HMAC flag set)                        |
// |------------------------------------------------------------------------------
// | 32..|                      Optional Fields and Payload                      |
// |------------------------------------------------------------------------------

// little endian used for multi-byte ints
var endian = binary.LittleEndian

// Seqno is a sequence number.
type Seqno uint32

// ctoken is a conn token
type ctoken uint64

// magic bytes
var magic = []byte{0x14, 0xa7, 0x5b}

// packet flags
type flags byte

func (f flags) set(fset flags) flags {
	return f | fset
}

func (f flags) clear(fcl flags) flags {
	return f &^ fcl
}

func (f flags) isset(fl flags) bool {
	return f&fl != 0
}

const (
	// flOpen is set when opening a new conn, both in the initial request from
	// the client to the server, and in the reply from the server.
	flOpen flags = 1 << iota

	// flReply is set in all packets from the server to the client, and unset in
	// all packets from the client to the server.
	flReply

	// flClose is set when closing a conn, both in the final request from the
	// client to the server, and in the reply from the server.
	flClose

	// flHMAC is set if an HMAC hash is included (so we can tell the
	// difference between a missing and invalid HMAC).
	flHMAC
)

const flAll = flOpen | flReply | flClose | flHMAC

// field indexes
const (
	fMagic fidx = iota
	fFlags
	fHMAC
	fConnToken
	fSeqno
	fRWall
	fRMono
	fMWall
	fMMono
	fSWall
	fSMono
)

const fcount = fSMono + 1

const foptidx = fHMAC

// field capacities (sync with field constants)
var fcaps = []int{3, 1, md5.Size, 8, 4, 8, 8, 8, 8, 8, 8}

// field index definitions
var finit = []fidx{fMagic, fFlags}

var finitHMAC = []fidx{fMagic, fFlags, fHMAC}

var fopenReply = []fidx{fMagic, fFlags, fConnToken}

var fcloseRequest = []fidx{fMagic, fFlags, fConnToken}

var fechoRequest = []fidx{fMagic, fFlags, fConnToken, fSeqno}

var fechoReply = []fidx{fMagic, fFlags, fConnToken, fSeqno}

// minHeaderLen is the minimum header length (set in init).
var minHeaderLen int

// maxHeaderLen is the maximum header length (set in init).
var maxHeaderLen int

func init() {
	for i := fidx(0); i < fcount; i++ {
		if i < foptidx {
			minHeaderLen += fcaps[i]
		}
		maxHeaderLen += fcaps[i]
	}
}

func newFields() []field {
	f := make([]field, fcount)
	for i := fidx(0); i < fcount; i++ {
		f[i].cap = fcaps[i]
	}
	return f
}

// Decorations for Time

func (t *Time) setWallFromBytes(b []byte) {
	t.Wall = int64(endian.Uint64(b[:]))
}

func (t *Time) setMonoFromBytes(b []byte) {
	t.Mono = time.Duration(endian.Uint64(b[:]))
}

func (t *Time) wallToBytes(b []byte) {
	endian.PutUint64(b[:], uint64(t.Wall))
}

func (t *Time) monoToBytes(b []byte) {
	endian.PutUint64(b[:], uint64(t.Mono))
}

// Packet struct and construction/set methods

type packet struct {
	*fbuf
	md5Hash hash.Hash
	hmacKey []byte
}

func newPacket(tlen int, cap int, hmacKey []byte) *packet {
	p := &packet{fbuf: newFbuf(newFields(), tlen, cap)}
	if len(hmacKey) > 0 {
		p.setFields(finitHMAC, true)
		p.md5Hash = hmac.New(md5.New, hmacKey)
		p.hmacKey = hmacKey
	} else {
		p.setFields(finit, true)
	}
	p.set(fMagic, magic)
	return p
}

func (p *packet) reset() error {
	if p.md5Hash != nil {
		p.setFields(finitHMAC, true)
	} else {
		p.setFields(finit, true)
	}
	flen, _ := p.sumFields()
	p.buf = p.buf[:flen]
	p.tlen = 0
	return p.fbuf.validate()
}

func (p *packet) readReset(n int) error {
	if p.md5Hash != nil {
		p.setFields(finitHMAC, false)
	} else {
		p.setFields(finit, false)
	}
	p.buf = p.buf[:n]
	p.tlen = n
	if err := p.fbuf.validate(); err != nil {
		return err
	}
	return p.validate()
}

func (p *packet) readTo() []byte {
	p.buf = p.buf[:cap(p.buf)]
	return p.buf
}

func (p *packet) validate() error {
	// magic
	if !bytesEqual(p.get(fMagic), magic) {
		return Errorf(BadMagic, "bad magic: %x != %x", p.get(fMagic), magic)
	}

	// flags
	if p.flags()&flOpen != 0 && p.flags()&flClose != 0 {
		return Errorf(OpenCloseBothSet, "open and close flags are both set")
	}
	if p.flags() > flAll {
		return Errorf(InvalidFlagBitsSet, "invalid flag bits set (%x)", p.flags())
	}

	// if there's a midpoint timestamp, there should be nothing else
	if p.hasMidpointStamp() && (p.hasReceiveStamp() || p.hasSendStamp()) {
		return Errorf(NonexclusiveMidpointTStamp, "non-exclusive midpoint timestamp")
	}

	// clock mode should be consistent for both stamps, as of now
	if p.hasReceiveStamp() && p.hasSendStamp() {
		rclock := clockFromBools(p.isset(fRWall), p.isset(fRMono))
		sclock := clockFromBools(p.isset(fSWall), p.isset(fSMono))
		if sclock != rclock {
			return Errorf(InconsistentClocks,
				"inconsistent clock mode between send and receive timestamps, %s != %s",
				sclock, rclock)
		}
	}

	// validate HMAC
	if p.md5Hash != nil {
		if p.flags()&flHMAC == 0 {
			return Errorf(NoHMAC, "no HMAC present")
		}
		p.addFields([]fidx{fHMAC}, false)
		y := make([]byte, md5.Size)
		copy(y[:], p.get(fHMAC))
		p.zero(fHMAC)
		p.md5Hash.Reset()
		p.md5Hash.Write(p.bytes())
		x := p.md5Hash.Sum(nil)
		if !hmac.Equal(y, x) {
			return Errorf(BadHMAC, "invalid HMAC: %x != %x", y, x)
		}
	} else if p.flags()&flHMAC != 0 {
		return Errorf(UnexpectedHMAC, "unexpected HMAC present")
	}
	return nil
}

// flags

func (p *packet) flags() flags {
	return flags(p.getb(fFlags))
}

func (p *packet) setFlagBits(f flags) {
	p.setb(fFlags, byte(p.flags().set(f)))
}

func (p *packet) clearFlagBits(f flags) {
	p.setb(fFlags, byte(p.flags().clear(f)))
}

// Reply

func (p *packet) reply() bool {
	return p.flags()&flReply != 0
}

func (p *packet) setReply(r bool) {
	if r {
		p.setFlagBits(flReply)
	} else {
		p.clearFlagBits(flReply)
	}
}

// Token

func (p *packet) ctoken() ctoken {
	return ctoken(endian.Uint64(p.get(fConnToken)))
}

func (p *packet) setConnToken(ctoken ctoken) {
	endian.PutUint64(p.setTo(fConnToken), uint64(ctoken))
}

// Sequence Number

func (p *packet) seqno() Seqno {
	return Seqno(endian.Uint32(p.get(fSeqno)))
}

func (p *packet) setSeqno(seqno Seqno) {
	endian.PutUint32(p.setTo(fSeqno), uint32(seqno))
}

// Timestamps

func (p *packet) timestamp() (ts Timestamp) {
	tget := func(wf fidx, mf fidx, t *Time) {
		wb := p.get(wf)
		if len(wb) > 0 {
			t.setWallFromBytes(wb)
		}
		mb := p.get(mf)
		if len(mb) > 0 {
			t.setMonoFromBytes(mb)
		}
	}

	tget(fRWall, fRMono, &ts.Receive)
	tget(fMWall, fMMono, &ts.Receive)
	tget(fMWall, fMMono, &ts.Send)
	tget(fSWall, fSMono, &ts.Send)

	return
}

func (p *packet) setTimestamp(ts Timestamp) {
	tset := func(t *Time, wf fidx, mf fidx) {
		if t.Wall != 0 {
			t.wallToBytes(p.setTo(wf))
		}
		if t.Mono != 0 {
			t.monoToBytes(p.setTo(mf))
		}
	}

	if ts.IsMidpoint() {
		tset(&ts.Receive, fMWall, fMMono)
		return
	}
	if !ts.Receive.IsZero() {
		tset(&ts.Receive, fRWall, fRMono)
	}
	if !ts.Send.IsZero() {
		tset(&ts.Send, fSWall, fSMono)
	}
}

func (p *packet) hasReceiveStamp() bool {
	return p.isset(fRWall) || p.isset(fRMono)
}

func (p *packet) hasMidpointStamp() bool {
	return p.isset(fMWall) || p.isset(fMMono)
}

func (p *packet) hasSendStamp() bool {
	return p.isset(fSWall) || p.isset(fSMono)
}

func (p *packet) clock() Clock {
	c := Clock(0)
	if p.isset(fRWall) || p.isset(fSWall) || p.isset(fMWall) {
		c |= Wall
	}
	if p.isset(fRMono) || p.isset(fSMono) || p.isset(fMMono) {
		c |= Monotonic
	}
	return c
}

func (p *packet) stampAt() (a StampAt) {
	if p.isset(fMWall) || p.isset(fMMono) {
		a = AtMidpoint
		return
	}
	if p.isset(fRWall) || p.isset(fRMono) {
		a |= AtReceive
	}
	if p.isset(fSWall) || p.isset(fSMono) {
		a |= AtSend
	}
	return
}

func (p *packet) stampZeroes(at StampAt, c Clock) {
	zts := func(a StampAt, wf fidx, mf fidx) {
		if at&a != 0 {
			if c&Wall != 0 {
				p.zero(wf)
			}
			if c&Monotonic != 0 {
				p.zero(mf)
			}
		} else {
			p.remove(wf)
			p.remove(mf)
		}
	}

	zts(AtReceive, fRWall, fRMono)
	zts(AtMidpoint, fMWall, fMMono)
	zts(AtSend, fSWall, fSMono)
}

func (p *packet) addTimestampFields(at StampAt, c Clock, setLen bool) {
	tfs := make([]fidx, 0, 4)
	atf := func(a StampAt, wf fidx, mf fidx) {
		if at&a != 0 {
			if c&Wall != 0 {
				tfs = append(tfs, wf)
			}
			if c&Monotonic != 0 {
				tfs = append(tfs, mf)
			}
		}
	}

	atf(AtReceive, fRWall, fRMono)
	atf(AtMidpoint, fMWall, fMMono)
	atf(AtSend, fSWall, fSMono)
	p.addFields(tfs, setLen)
}

func (p *packet) removeTimestamps() {
	p.remove(fRWall)
	p.remove(fRMono)
	p.remove(fMWall)
	p.remove(fMMono)
	p.remove(fSWall)
	p.remove(fSMono)
}

// HMAC

func (p *packet) updateHMAC() {
	if p.md5Hash != nil {
		// calculate and set hmac, with zeroed hmac field
		p.setFlagBits(flHMAC)
		p.zero(fHMAC)
		p.md5Hash.Reset()
		p.md5Hash.Write(p.buf)
		mac := p.md5Hash.Sum(nil)
		p.set(fHMAC, mac)
	} else if p.isset(fHMAC) {
		// clear field and flags
		p.clearFlagBits(flHMAC)
		p.remove(fHMAC)
	}
}

// Payload

func (p *packet) readPayload(r io.Reader) error {
	_, err := io.ReadFull(r, p.payload())
	return err
}
