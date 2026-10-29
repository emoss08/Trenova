package factoringtransport_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/factoringtransport"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func file(name string, size int) factoringsubmissionports.TransportFile {
	return factoringsubmissionports.TransportFile{
		Name:        name,
		ContentType: "application/pdf",
		Content:     []byte(strings.Repeat("x", size)),
	}
}

func TestPackAttachments(t *testing.T) {
	t.Parallel()

	parts, err := factoringtransport.PackAttachments([]factoringsubmissionports.TransportFile{
		file(
			"FS1.pdf",
			40,
		), file("FS1.csv", 10), file("A.pdf", 60), file("B.pdf", 30), file("C.pdf", 90),
	}, 100)
	require.NoError(t, err)
	require.Len(t, parts, 3)
	assert.Len(t, parts[0], 2)
	assert.Equal(t, "A.pdf", parts[1][0].Name)
	assert.Equal(t, "B.pdf", parts[1][1].Name)
	assert.Equal(t, "C.pdf", parts[2][0].Name)

	_, err = factoringtransport.PackAttachments(
		[]factoringsubmissionports.TransportFile{file("huge.pdf", 101)},
		100,
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "huge.pdf")

	_, err = factoringtransport.PackAttachments(nil, 100)
	require.Error(t, err)
}

type fakeEmail struct {
	emailports.EmailService
	sent []*emailports.SendEmailRequest
}

func (f *fakeEmail) Send(
	_ context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
	f.sent = append(f.sent, req)
	return &email.Message{ID: pulid.MustNew("emsg_")}, nil
}

func submission() *factoringsubmission.Submission {
	return &factoringsubmission.Submission{ID: pulid.MustNew("facsub_"), Number: "FS2610000001"}
}

func TestEmailSendsOnePartWithKeyedParts(t *testing.T) {
	t.Parallel()

	fake := &fakeEmail{}
	transport := factoringtransport.NewEmail(factoringtransport.EmailParams{Email: fake})
	var rendered []string
	result, err := transport.Send(t.Context(), &factoringsubmissionports.TransportSendRequest{
		Submission: submission(),
		Transport: &factoringsubmission.Transport{
			Method:          factoringsubmission.TransportMethodEmail,
			EmailRecipients: []string{"schedules@factor.test"},
		},
		Files: []factoringsubmissionports.TransportFile{file("FS.pdf", 10), file("FS.csv", 5)},
		Render: func(part, parts int, attachments []string) (*factoringsubmissionports.TransportMessage, error) {
			rendered = append(rendered, strings.Join(attachments, ","))
			return &factoringsubmissionports.TransportMessage{
				Subject: "Schedule",
				Text:    "body",
			}, nil
		},
		IdempotencyKey: "facsub-1-send-1",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Parts)
	require.Len(t, fake.sent, 1)
	assert.Equal(t, "facsub-1-send-1-part-1", fake.sent[0].IdempotencyKey)
	assert.Equal(t, []string{"schedules@factor.test"}, fake.sent[0].To)
	assert.Len(t, fake.sent[0].Attachments, 2)
	assert.Equal(t, []string{"FS.pdf,FS.csv"}, rendered)
}

type fakeRemote struct {
	written map[string]int
	statErr error
}

func (f *fakeRemote) WriteFile(remotePath string, contents []byte) error {
	f.written[remotePath] = len(contents)
	return nil
}

func (f *fakeRemote) Stat(string) error { return f.statErr }
func (f *fakeRemote) Close() error      { return nil }

func sftpTransport() *factoringsubmission.Transport {
	return &factoringsubmission.Transport{
		Method:           factoringsubmission.TransportMethodSftp,
		SftpHost:         "sftp.factor.test",
		SftpPort:         22,
		SftpUsername:     "carrier",
		SftpAuthMethod:   factoringsubmission.SftpAuthPassword,
		SftpKnownHostKey: "sftp.factor.test ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
		SftpDirectory:    "/inbound",
	}
}

func TestSftpUploadsEveryFileUnderTheScheduleNumber(t *testing.T) {
	t.Parallel()

	remote := &fakeRemote{written: map[string]int{}}
	var dialed sftp.Config
	transport := factoringtransport.NewSftpWith(
		func(_ context.Context, cfg *sftp.Config) (factoringtransport.RemoteClient, error) {
			dialed = *cfg
			return remote, nil
		},
		func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("93.184.215.14")}, nil },
	)
	result, err := transport.Send(t.Context(), &factoringsubmissionports.TransportSendRequest{
		Submission: submission(),
		Transport:  sftpTransport(),
		Secret:     "hunter2",
		Files: []factoringsubmissionports.TransportFile{
			file("FS2610000001.pdf", 3),
			file("../INV-1.pdf", 4),
		},
		IdempotencyKey: "facsub-1-send-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "/inbound/FS2610000001", result.Reference)
	assert.Equal(t, map[string]int{
		"/inbound/FS2610000001/FS2610000001.pdf": 3,
		"/inbound/FS2610000001/-INV-1.pdf":       4,
	}, remote.written)
	assert.Equal(t, "93.184.215.14", dialed.Host)
	assert.Equal(t, sftp.AuthModePassword, dialed.AuthMode)
	assert.Equal(t, "hunter2", dialed.Password)
}

func TestSftpRefusesPrivateHostsAndMissingSecrets(t *testing.T) {
	t.Parallel()

	dialCalled := false
	transport := factoringtransport.NewSftpWith(
		func(context.Context, *sftp.Config) (factoringtransport.RemoteClient, error) {
			dialCalled = true
			return nil, errors.New("should not dial")
		},
		func(context.Context, string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("93.184.215.14"), net.ParseIP("10.0.0.5")}, nil
		},
	)
	err := transport.Test(t.Context(), &factoringsubmissionports.TransportTestRequest{
		Transport: sftpTransport(),
		Secret:    "hunter2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private or reserved")

	err = transport.Test(
		t.Context(),
		&factoringsubmissionports.TransportTestRequest{Transport: sftpTransport()},
	)
	require.Error(t, err)
	assert.False(t, dialCalled)
}

func TestSftpTestReportsAnUnreadableDirectory(t *testing.T) {
	t.Parallel()

	transport := factoringtransport.NewSftpWith(
		func(context.Context, *sftp.Config) (factoringtransport.RemoteClient, error) {
			return &fakeRemote{statErr: errors.New("permission denied")}, nil
		},
		func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("93.184.215.14")}, nil },
	)
	err := transport.Test(t.Context(), &factoringsubmissionports.TransportTestRequest{
		Transport: sftpTransport(),
		Secret:    "hunter2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/inbound")
}

func TestRemoteName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "file", factoringtransport.RemoteName(".."))
	assert.Equal(t, "a-b.pdf", factoringtransport.RemoteName("a/b.pdf"))
}
