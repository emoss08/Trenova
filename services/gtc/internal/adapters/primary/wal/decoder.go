package wal

import (
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
)

type Decoder struct {
	relations          map[uint32]*pglogrepl.RelationMessageV2
	typeMap            *pgtype.Map
	inStream           bool
	currentTransaction transactionState
}

type transactionState struct {
	lsn        pglogrepl.LSN
	commitLSN  pglogrepl.LSN
	xid        uint32
	commitTime time.Time
	records    []domain.SourceRecord
}

func NewDecoder() *Decoder {
	return &Decoder{
		relations: make(map[uint32]*pglogrepl.RelationMessageV2),
		typeMap:   pgtype.NewMap(),
	}
}

type DecodeResult struct {
	Transaction *domain.TransactionRecords
	LSN         pglogrepl.LSN
}

func (d *Decoder) Decode(rawMsg pgproto3.BackendMessage) (*DecodeResult, error) {
	copyData, ok := rawMsg.(*pgproto3.CopyData)
	if !ok {
		return &DecodeResult{}, nil
	}

	switch copyData.Data[0] {
	case pglogrepl.PrimaryKeepaliveMessageByteID:
		pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(copyData.Data[1:])
		if err != nil {
			return nil, err
		}
		return &DecodeResult{LSN: pkm.ServerWALEnd}, nil

	case pglogrepl.XLogDataByteID:
		xld, err := pglogrepl.ParseXLogData(copyData.Data[1:])
		if err != nil {
			return nil, err
		}
		tx, err := d.decodeWALData(xld.WALData, xld.WALStart)
		if err != nil {
			return nil, err
		}
		return &DecodeResult{Transaction: tx, LSN: xld.WALStart}, nil
	}

	return &DecodeResult{}, nil
}

func (d *Decoder) decodeWALData(walData []byte, lsn pglogrepl.LSN) (*domain.TransactionRecords, error) {
	logicalMsg, err := pglogrepl.ParseV2(walData, d.inStream)
	if err != nil {
		return nil, err
	}

	switch msg := logicalMsg.(type) {
	case *pglogrepl.BeginMessage:
		d.currentTransaction = transactionState{
			lsn:        lsn,
			xid:        msg.Xid,
			commitTime: msg.CommitTime.UTC(),
			records:    make([]domain.SourceRecord, 0, 8),
		}

	case *pglogrepl.CommitMessage:
		d.currentTransaction.commitLSN = msg.CommitLSN
		d.currentTransaction.commitTime = msg.CommitTime.UTC()
		tx := &domain.TransactionRecords{
			LSN:           d.currentTransaction.lsn.String(),
			CommitLSN:     msg.CommitLSN.String(),
			TransactionID: d.currentTransaction.xid,
			Timestamp:     d.currentTransaction.commitTime,
			Records:       d.currentTransaction.records,
		}
		d.currentTransaction = transactionState{}
		return tx, nil

	case *pglogrepl.RelationMessageV2:
		d.relations[msg.RelationID] = msg

	case *pglogrepl.InsertMessageV2:
		rel := d.relations[msg.RelationID]
		if rel == nil {
			return nil, nil
		}
		d.appendRecord(domain.SourceRecord{
			Operation: domain.OperationInsert,
			Schema:    rel.Namespace,
			Table:     rel.RelationName,
			NewData:   d.decodeTuple(msg.Tuple, rel),
			Metadata:  d.metadata(lsn, msg.Xid),
		})

	case *pglogrepl.UpdateMessageV2:
		rel := d.relations[msg.RelationID]
		if rel == nil {
			return nil, nil
		}
		event := domain.SourceRecord{
			Operation: domain.OperationUpdate,
			Schema:    rel.Namespace,
			Table:     rel.RelationName,
			NewData:   d.decodeTuple(msg.NewTuple, rel),
			Metadata:  d.metadata(lsn, msg.Xid),
		}
		if msg.OldTuple != nil {
			event.OldData = d.decodeTuple(msg.OldTuple, rel)
		}
		d.appendRecord(event)

	case *pglogrepl.DeleteMessageV2:
		rel := d.relations[msg.RelationID]
		if rel == nil {
			return nil, nil
		}
		d.appendRecord(domain.SourceRecord{
			Operation: domain.OperationDelete,
			Schema:    rel.Namespace,
			Table:     rel.RelationName,
			OldData:   d.decodeTuple(msg.OldTuple, rel),
			Metadata:  d.metadata(lsn, msg.Xid),
		})

	case *pglogrepl.TruncateMessageV2:
		for _, relID := range msg.RelationIDs {
			if rel, ok := d.relations[relID]; ok {
				d.appendRecord(domain.SourceRecord{
					Operation: domain.OperationTruncate,
					Schema:    rel.Namespace,
					Table:     rel.RelationName,
					Metadata:  d.metadata(lsn, msg.Xid),
				})
			}
		}

	case *pglogrepl.StreamStartMessageV2:
		d.inStream = true

	case *pglogrepl.StreamStopMessageV2:
		d.inStream = false
	}

	return nil, nil
}

func (d *Decoder) appendRecord(record domain.SourceRecord) {
	if d.currentTransaction.records == nil {
		d.currentTransaction.records = make([]domain.SourceRecord, 0, 8)
	}
	d.currentTransaction.records = append(d.currentTransaction.records, record)
}

func (d *Decoder) metadata(lsn pglogrepl.LSN, xid uint32) domain.EventMetadata {
	if xid == 0 {
		xid = d.currentTransaction.xid
	}

	return domain.EventMetadata{
		LSN:           lsn.String(),
		TransactionID: xid,
		Timestamp:     d.currentTransaction.commitTime,
	}
}

func (d *Decoder) decodeTuple(
	tuple *pglogrepl.TupleData,
	rel *pglogrepl.RelationMessageV2,
) map[string]any {
	if tuple == nil {
		return nil
	}

	values := make(map[string]any, len(tuple.Columns))
	for idx, col := range tuple.Columns {
		colName := rel.Columns[idx].Name
		switch col.DataType {
		case 'n':
			values[colName] = nil
		case 'u':
			values[colName] = "(unchanged-toast)"
		case 't':
			val, err := d.decodeTextColumn(col.Data, rel.Columns[idx].DataType)
			if err != nil {
				values[colName] = string(col.Data)
			} else {
				values[colName] = val
			}
		}
	}

	return values
}

func (d *Decoder) decodeTextColumn(data []byte, dataType uint32) (any, error) {
	if dt, ok := d.typeMap.TypeForOID(dataType); ok {
		return dt.Codec.DecodeValue(d.typeMap, dataType, pgtype.TextFormatCode, data)
	}
	return string(data), nil
}
