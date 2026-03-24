package wal

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/jackc/pglogrepl"
)

func TestDecoderEmitsCommittedTransaction(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder()
	beginTime := time.Date(2026, 3, 20, 20, 12, 34, 0, time.UTC)

	if _, err := decoder.decodeWALData(encodeBeginMessage(pglogrepl.LSN(100), beginTime, 42), pglogrepl.LSN(90)); err != nil {
		t.Fatalf("decode begin: %v", err)
	}

	decoder.appendRecord(domain.SourceRecord{
		Operation: domain.OperationInsert,
		Schema:    "public",
		Table:     "shipments",
		NewData:   map[string]any{"id": "shp_1"},
	})

	tx, err := decoder.decodeWALData(
		encodeCommitMessage(pglogrepl.LSN(120), pglogrepl.LSN(121), beginTime.Add(time.Second)),
		pglogrepl.LSN(110),
	)
	if err != nil {
		t.Fatalf("decode commit: %v", err)
	}
	if tx == nil {
		t.Fatalf("expected committed transaction")
	}
	if tx.CommitLSN != "0/78" {
		t.Fatalf("unexpected commit lsn: %s", tx.CommitLSN)
	}
	if tx.TransactionID != 42 {
		t.Fatalf("unexpected xid: %d", tx.TransactionID)
	}
	if len(tx.Records) != 1 {
		t.Fatalf("expected one buffered record, got %d", len(tx.Records))
	}
	if !tx.Timestamp.Equal(beginTime.Add(time.Second)) {
		t.Fatalf("unexpected commit time: %s", tx.Timestamp)
	}
}

func encodeBeginMessage(finalLSN pglogrepl.LSN, commitTime time.Time, xid uint32) []byte {
	buf := make([]byte, 1+8+8+4)
	buf[0] = byte(pglogrepl.MessageTypeBegin)
	binary.BigEndian.PutUint64(buf[1:9], uint64(finalLSN))
	binary.BigEndian.PutUint64(buf[9:17], uint64(timeToPgMicros(commitTime)))
	binary.BigEndian.PutUint32(buf[17:21], xid)
	return buf
}

func encodeCommitMessage(commitLSN pglogrepl.LSN, endLSN pglogrepl.LSN, commitTime time.Time) []byte {
	buf := make([]byte, 1+1+8+8+8)
	buf[0] = byte(pglogrepl.MessageTypeCommit)
	buf[1] = 0
	binary.BigEndian.PutUint64(buf[2:10], uint64(commitLSN))
	binary.BigEndian.PutUint64(buf[10:18], uint64(endLSN))
	binary.BigEndian.PutUint64(buf[18:26], uint64(timeToPgMicros(commitTime)))
	return buf
}

func timeToPgMicros(t time.Time) int64 {
	const microsecFromUnixEpochToY2K = 946684800 * 1000000
	microsecSinceUnixEpoch := t.Unix()*1000000 + int64(t.Nanosecond())/1000
	return microsecSinceUnixEpoch - microsecFromUnixEpochToY2K
}
