package operationbits

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	Read   uint32 = 1 << 0
	Create uint32 = 1 << 1
	Update uint32 = 1 << 2
	Delete uint32 = 1 << 3
	Export uint32 = 1 << 4
	Import uint32 = 1 << 5

	Approve    uint32 = 1 << 8
	Reject     uint32 = 1 << 9
	Assign     uint32 = 1 << 10
	Unassign   uint32 = 1 << 11
	Archive    uint32 = 1 << 12
	Restore    uint32 = 1 << 13
	Submit     uint32 = 1 << 14
	Cancel     uint32 = 1 << 15
	Duplicate  uint32 = 1 << 16
	Close      uint32 = 1 << 17
	Lock       uint32 = 1 << 18
	Unlock     uint32 = 1 << 19
	Activate   uint32 = 1 << 20
	Reopen     uint32 = 1 << 21
	Pin        uint32 = 1 << 22
	Unpin      uint32 = 1 << 23
	Resolve    uint32 = 1 << 24
	Manage     uint32 = 1 << 25
	Post       uint32 = 1 << 26
	ApproveOwn uint32 = 1 << 27
)

var ByOperation = map[permission.Operation]uint32{
	permission.OpRead:       Read,
	permission.OpCreate:     Create,
	permission.OpUpdate:     Update,
	permission.OpDelete:     Delete,
	permission.OpExport:     Export,
	permission.OpImport:     Import,
	permission.OpApprove:    Approve,
	permission.OpReject:     Reject,
	permission.OpAssign:     Assign,
	permission.OpUnassign:   Unassign,
	permission.OpArchive:    Archive,
	permission.OpRestore:    Restore,
	permission.OpSubmit:     Submit,
	permission.OpCancel:     Cancel,
	permission.OpDuplicate:  Duplicate,
	permission.OpClose:      Close,
	permission.OpLock:       Lock,
	permission.OpUnlock:     Unlock,
	permission.OpActivate:   Activate,
	permission.OpReopen:     Reopen,
	permission.OpPin:        Pin,
	permission.OpUnpin:      Unpin,
	permission.OpResolve:    Resolve,
	permission.OpManage:     Manage,
	permission.OpPost:       Post,
	permission.OpApproveOwn: ApproveOwn,
}

func Mask(ops []permission.Operation) uint32 {
	var bits uint32
	for _, op := range ops {
		if bit, ok := ByOperation[op]; ok {
			bits |= bit
		}
	}
	return bits
}
