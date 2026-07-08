package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/shared/as2"
)

type runner struct {
	apiBase      string
	simBase      string
	client       *http.Client
	csrfToken    string
	failures     int
	trenovaAS2ID string
	simAS2ID     string
}

func main() {
	apiBase := flag.String("api", "http://localhost:8080/api/v1", "Trenova API base URL")
	simBase := flag.String("sim", "http://localhost:9210", "EDI partner simulator base URL")
	email := flag.String("email", "admin@trenova.app", "login email")
	password := flag.String("password", "admin123!", "login password")
	inboundURL := flag.String(
		"inbound",
		"http://localhost:8080/api/v1/edi/as2/inbound/",
		"Trenova AS2 inbound URL the simulator posts to",
	)
	flag.Parse()

	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("FATAL: cookie jar:", err)
		os.Exit(1)
	}
	r := &runner{
		apiBase: strings.TrimRight(*apiBase, "/"),
		simBase: strings.TrimRight(*simBase, "/"),
		client:  &http.Client{Jar: jar, Timeout: 60 * time.Second},
	}

	fmt.Println("=== Trenova EDI end-to-end run ===")
	if !r.run(*email, *password, *inboundURL) || r.failures > 0 {
		fmt.Printf("\nRESULT: FAILED (%d failed step(s))\n", r.failures)
		os.Exit(1)
	}
	fmt.Println("\nRESULT: ALL STEPS PASSED")
}

//nolint:funlen // The scenario is intentionally one linear script.
func (r *runner) run(email, password, inboundURL string) bool {
	// 1. Preconditions
	var simIdentity struct {
		AS2ID          string `json:"as2Id"`
		RemoteAS2ID    string `json:"remoteAs2Id"`
		CertificatePEM string `json:"certificatePem"`
	}
	if !r.step("simulator is reachable", func() error {
		return r.getJSON(r.simBase+"/control/identity", &simIdentity)
	}) {
		return false
	}
	suffix := time.Now().Format("150405")
	r.simAS2ID = "SIM" + suffix
	r.trenovaAS2ID = "TRN" + suffix

	// 2. Login + role activation
	var login struct {
		CSRFToken       string `json:"csrfToken"`
		AuthorizedRoles []struct {
			ID string `json:"id"`
		} `json:"authorizedRoles"`
		ActiveRoles []struct {
			ID string `json:"id"`
		} `json:"activeRoles"`
	}
	if !r.step("login as "+email, func() error {
		return r.postJSON(r.apiBase+"/auth/login", map[string]string{
			"emailAddress": email,
			"password":     password,
		}, &login)
	}) {
		return false
	}
	r.csrfToken = login.CSRFToken

	if len(login.ActiveRoles) == 0 {
		roleIDs := make([]string, 0, len(login.AuthorizedRoles))
		for _, role := range login.AuthorizedRoles {
			roleIDs = append(roleIDs, role.ID)
		}
		if !r.step(fmt.Sprintf("activate %d session role(s)", len(roleIDs)), func() error {
			return r.postJSON(r.apiBase+"/auth/session/roles/activate", map[string]any{
				"roleIds": roleIDs,
			}, nil)
		}) {
			return false
		}
	}

	// 3. Trenova-side AS2 identity + simulator wiring
	trenova, err := sim.NewIdentity("trenova.e2e.local")
	if err != nil {
		fmt.Println("FATAL: generate Trenova identity:", err)
		return false
	}
	r.step("configure simulator (certificate + unique AS2 identity)", func() error {
		return r.postJSON(r.simBase+"/control/partner", map[string]any{
			"certificatePem": trenova.CertificatePEM,
			"inboundUrl":     inboundURL,
			"autoAck":        true,
			"as2Id":          r.simAS2ID,
			"remoteAs2Id":    r.trenovaAS2ID,
		}, nil)
	})
	r.step("reset simulator state", func() error {
		return r.postJSON(r.simBase+"/control/reset", map[string]any{}, nil)
	})

	// 4. Partner
	partnerCode := "SIM" + suffix
	var partner struct {
		ID string `json:"id"`
	}
	if !r.step("create external partner "+partnerCode, func() error {
		return r.postJSON(r.apiBase+"/edi/partners/", map[string]any{
			"kind":               "External",
			"status":             "Active",
			"code":               partnerCode,
			"name":               "EDI Partner Simulator " + suffix,
			"country":            "US",
			"contactName":        "Simulator Operator",
			"contactEmail":       "sim@partner.example",
			"timezone":           "America/Chicago",
			"enabledForInbound":  true,
			"enabledForOutbound": true,
			"settings":           map[string]any{},
		}, &partner)
	}) {
		return false
	}

	// 5. AS2 communication profile
	var profile struct {
		ID string `json:"id"`
	}
	if !r.step("create AS2 communication profile", func() error {
		return r.postJSON(r.apiBase+"/edi/communication-profiles/", map[string]any{
			"ediPartnerId": partner.ID,
			"method":       "AS2",
			"status":       "Active",
			"name":         "Simulator AS2 " + suffix,
			"config": map[string]any{
				"localAS2Id":                r.trenovaAS2ID,
				"partnerAS2Id":              r.simAS2ID,
				"endpointUrl":               r.simBase + "/as2",
				"mdnMode":                   "sync",
				"signingAlgorithm":          "sha256",
				"encryptionAlgorithm":       "aes256-cbc",
				"compressionAlgorithm":      "none",
				"localCertificate":          trenova.CertificatePEM,
				"partnerSigningCertificate": simIdentity.CertificatePEM,
				"requireSignedInbound":      "auto",
				"requireEncryptedInbound":   "auto",
				"isaSenderQualifier":        "ZZ",
				"isaSenderId":               r.trenovaAS2ID,
				"isaReceiverQualifier":      "ZZ",
				"isaReceiverId":             r.simAS2ID,
				"gsSenderId":                r.trenovaAS2ID,
				"gsReceiverId":              r.simAS2ID,
				"x12Version":                "004010",
				"environment":               "test",
				"acknowledgmentPreference":  "997",
			},
			"secrets": map[string]string{
				"privateKey": trenova.KeyPEM,
			},
		}, &profile)
	}) {
		return false
	}

	// 6. Test connection (WS3.4)
	r.step("test-connection reports success", func() error {
		var result struct {
			Success bool `json:"success"`
			Checks  []struct {
				Name    string `json:"name"`
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"checks"`
		}
		if err := r.postJSON(
			r.apiBase+"/edi/communication-profiles/"+profile.ID+"/test-connection/",
			map[string]any{},
			&result,
		); err != nil {
			return err
		}
		for _, check := range result.Checks {
			fmt.Printf("      · %-28s %-8s %s\n", check.Name, check.Status, check.Message)
		}
		if !result.Success {
			return fmt.Errorf("connection test reported failure")
		}
		return nil
	})

	// 7. Document profile (auto-provisions the base 204 template)
	var documentProfile struct {
		ID string `json:"id"`
	}
	if !r.step("create outbound 204 document profile", func() error {
		return r.postJSON(r.apiBase+"/edi/document-profiles/", map[string]any{
			"ediPartnerId":   partner.ID,
			"name":           "Simulator 204 Outbound " + suffix,
			"status":         "Active",
			"validationMode": "WarnOnly",
			"envelope": map[string]any{
				"interchangeSenderQualifier":   "ZZ",
				"interchangeSenderId":          r.trenovaAS2ID,
				"interchangeReceiverQualifier": "ZZ",
				"interchangeReceiverId":        r.simAS2ID,
				"applicationSenderCode":        r.trenovaAS2ID,
				"applicationReceiverCode":      r.simAS2ID,
				"interchangeUsageIndicator":    "T",
				"elementSeparator":             "*",
				"segmentTerminator":            "~",
				"componentSeparator":           ">",
				"repetitionSeparator":          "^",
			},
			"acknowledgment": map[string]any{
				"expected":     true,
				"type":         "997",
				"slaInMinutes": 240,
			},
			"partnerSettings": map[string]any{
				"carrier": map[string]any{"scac": "SIML"},
			},
		}, &documentProfile)
	}) {
		return false
	}

	// 8. Pick a seeded shipment
	var shipments struct {
		Results []struct {
			ID        string `json:"id"`
			ProNumber string `json:"proNumber"`
		} `json:"results"`
	}
	if !r.step("find a seeded shipment", func() error {
		if err := r.getJSON(r.apiBase+"/shipments/?limit=1", &shipments); err != nil {
			return err
		}
		if len(shipments.Results) == 0 {
			return fmt.Errorf("no shipments found — run `task db-seed` first")
		}
		fmt.Printf(
			"      · using shipment %s (%s)\n",
			shipments.Results[0].ProNumber,
			shipments.Results[0].ID,
		)
		return nil
	}) {
		return false
	}
	shipmentID := shipments.Results[0].ID

	// 9. Generate the outbound 204
	var message struct {
		ID             string `json:"id"`
		TransactionSet string `json:"transactionSet"`
	}
	if !r.step("generate outbound 204 for the shipment", func() error {
		return r.postJSON(r.apiBase+"/edi/documents/generate/", map[string]any{
			"partnerDocumentProfileId": documentProfile.ID,
			"ediPartnerId":             partner.ID,
			"shipmentId":               shipmentID,
			"transactionSet":           "204",
			"direction":                "Outbound",
		}, &message)
	}) {
		return false
	}

	// 10. Delivery via Temporal + AS2 → simulator
	r.step("message delivers over AS2 (deliveryStatus=Sent)", func() error {
		return r.pollMessage(message.ID, 90*time.Second, func(m map[string]any) (bool, error) {
			status, _ := m["deliveryStatus"].(string)
			switch status {
			case "Sent":
				return true, nil
			case "Failed", "DeadLettered":
				return false, fmt.Errorf("delivery ended in %s: %v", status, m["deliveryLastError"])
			default:
				return false, nil
			}
		})
	})

	r.step("simulator received the signed+encrypted 204", func() error {
		var received struct {
			Received []struct {
				Signed    bool             `json:"signed"`
				Encrypted bool             `json:"encrypted"`
				Envelope  *sim.X12Envelope `json:"envelope"`
			} `json:"received"`
		}
		if err := r.getJSON(r.simBase+"/control/received", &received); err != nil {
			return err
		}
		for _, doc := range received.Received {
			if doc.Envelope != nil && doc.Envelope.TransactionSet == "204" {
				if !doc.Signed || !doc.Encrypted {
					return fmt.Errorf(
						"204 arrived unsigned/unencrypted (signed=%v encrypted=%v)",
						doc.Signed,
						doc.Encrypted,
					)
				}
				fmt.Printf("      · ISA %s / GS %s / ST %s\n",
					doc.Envelope.InterchangeControlNumber,
					doc.Envelope.GroupControlNumber,
					doc.Envelope.TransactionControlNumber,
				)
				return nil
			}
		}
		return fmt.Errorf("simulator has not received a 204")
	})

	// 11. 997 reconciliation (simulator auto-acks)
	r.step("997 reconciles the message (ackStatus=Accepted)", func() error {
		return r.pollMessage(message.ID, 90*time.Second, func(m map[string]any) (bool, error) {
			status, _ := m["ackStatus"].(string)
			switch status {
			case "Accepted":
				return true, nil
			case "Rejected", "Failed":
				return false, fmt.Errorf(
					"acknowledgment ended in %s: %v",
					status,
					m["ackLastError"],
				)
			default:
				return false, nil
			}
		})
	})

	// 12. WS0.2 security: unsigned/unencrypted inbound must be rejected
	r.step("unsigned inbound AS2 payload is rejected (WS0.2)", func() error {
		payload := sim.BuildLoadTender204(sim.BuildLoadTenderInput{
			SenderID:      r.simAS2ID,
			ReceiverID:    r.trenovaAS2ID,
			ControlNumber: time.Now().Unix() % 1000000,
			ShipmentID:    "UNSIGNED1",
		})
		built, err := as2.BuildMessage(&as2.BuildMessageOptions{
			From:    r.simAS2ID,
			To:      r.trenovaAS2ID,
			Payload: []byte(payload),
		})
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			inboundURL,
			bytes.NewReader(built.Body),
		)
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", built.ContentType)
		for key, values := range built.Headers {
			for _, value := range values {
				request.Header.Set(key, value)
			}
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		mdn, err := as2.ParseMDN(response.Header.Get("Content-Type"), body, nil)
		if err != nil {
			return fmt.Errorf("parse MDN: %w", err)
		}
		if mdn.Processed() {
			return fmt.Errorf("unsigned payload was ACCEPTED — the WS0.2 gate is not enforcing")
		}
		fmt.Printf("      · rejected with disposition %q\n", mdn.Disposition)
		return nil
	})

	// 13. Inbound 204 tender from the partner
	var tender struct {
		ShipmentID string `json:"shipmentId"`
		Record     struct {
			MDNStatus string `json:"mdnStatus"`
			Error     string `json:"error"`
		} `json:"record"`
	}
	r.step("simulator sends an inbound 204 load tender", func() error {
		if err := r.postJSON(r.simBase+"/control/send-tender", map[string]any{}, &tender); err != nil {
			return err
		}
		if tender.Record.MDNStatus != "processed" {
			return fmt.Errorf(
				"inbound MDN status %q: %s",
				tender.Record.MDNStatus,
				tender.Record.Error,
			)
		}
		fmt.Printf("      · tender reference %s accepted with processed MDN\n", tender.ShipmentID)
		return nil
	})

	r.step("inbound file is staged and processed", func() error {
		deadline := time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			var files struct {
				Results []struct {
					Status         string `json:"status"`
					FailureReason  string `json:"failureReason"`
					EDIPartnerID   string `json:"ediPartnerId"`
					TransactionCnt int    `json:"transactionCount"`
				} `json:"results"`
			}
			if err := r.getJSON(
				r.apiBase+"/edi/inbound-files/?limit=10&partnerId="+partner.ID,
				&files,
			); err != nil {
				return err
			}
			for _, file := range files.Results {
				switch file.Status {
				case "Processed", "PartiallyProcessed":
					fmt.Printf("      · inbound file status %s\n", file.Status)
					if file.FailureReason != "" {
						fmt.Printf("      · warnings: %s\n", file.FailureReason)
					}
					return nil
				case "Quarantined":
					return fmt.Errorf("inbound file quarantined: %s", file.FailureReason)
				}
			}
			time.Sleep(2 * time.Second)
		}
		return fmt.Errorf("inbound file did not finish processing in time")
	})

	r.step("inbound transfer exists for the tender", func() error {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			var transfers struct {
				Results []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"results"`
			}
			if err := r.getJSON(
				r.apiBase+"/edi/transfers/?direction=inbound&limit=5",
				&transfers,
			); err != nil {
				return err
			}
			if len(transfers.Results) > 0 {
				fmt.Printf(
					"      · newest inbound transfer status %s\n",
					transfers.Results[0].Status,
				)
				return nil
			}
			time.Sleep(2 * time.Second)
		}
		return fmt.Errorf("no inbound transfer appeared")
	})

	// 14. Partner readiness reflects the finished setup (WS5)
	r.step("partner readiness endpoint responds", func() error {
		var readiness struct {
			Ready          bool `json:"ready"`
			CompletedCount int  `json:"completedCount"`
			TotalCount     int  `json:"totalCount"`
		}
		if err := r.getJSON(
			r.apiBase+"/edi/partners/"+partner.ID+"/readiness/",
			&readiness,
		); err != nil {
			return err
		}
		fmt.Printf(
			"      · readiness %d/%d complete\n",
			readiness.CompletedCount,
			readiness.TotalCount,
		)
		return nil
	})

	return true
}

func (r *runner) pollMessage(
	messageID string,
	timeout time.Duration,
	done func(map[string]any) (bool, error),
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var message map[string]any
		if err := r.getJSON(r.apiBase+"/edi/messages/"+messageID+"/", &message); err != nil {
			return err
		}
		finished, err := done(message)
		if err != nil {
			return err
		}
		if finished {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out after %s", timeout)
}

func (r *runner) step(name string, fn func() error) bool {
	fmt.Printf("→ %s\n", name)
	if err := fn(); err != nil {
		fmt.Printf("  ✗ FAIL: %v\n", err)
		r.failures++
		return false
	}
	fmt.Println("  ✓ ok")
	return true
}

func (r *runner) getJSON(url string, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return r.do(request, out)
}

func (r *runner) postJSON(url string, payload, out any) error {
	body, err := sonic.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if r.csrfToken != "" {
		request.Header.Set("X-CSRF-Token", r.csrfToken)
	}
	return r.do(request, out)
}

func (r *runner) do(request *http.Request, out any) error {
	response, err := r.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf(
			"HTTP %d from %s: %s",
			response.StatusCode,
			request.URL.Path,
			strings.TrimSpace(string(body)),
		)
	}
	if out == nil {
		return nil
	}
	if err := sonic.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s response: %w", request.URL.Path, err)
	}
	return nil
}
