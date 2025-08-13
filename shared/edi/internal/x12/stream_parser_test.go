package x12

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStreamParser_Basic(t *testing.T) {
	input := "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~" +
		"GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~" +
		"ST*204*0001~" +
		"B2**SCAC*ABC123*CC~" +
		"SE*3*0001~" +
		"GE*1*1~" +
		"IEA*1*000000001~"

	reader := strings.NewReader(input)
	delims := Delimiters{Element: '*', Component: '>', Segment: '~'}
	
	var segments []Segment
	parser := NewStreamParser(reader, delims, WithSegmentHandler(func(s Segment) error {
		segments = append(segments, s)
		return nil
	}))
	
	err := parser.Parse(context.Background())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if len(segments) != 7 {
		t.Errorf("Expected 7 segments, got %d", len(segments))
	}
	
	// Verify first segment
	if segments[0].Tag != "ISA" {
		t.Errorf("Expected first segment tag ISA, got %s", segments[0].Tag)
	}
	
	// Verify metrics
	segCount, _ := parser.GetMetrics()
	if segCount != 7 {
		t.Errorf("Expected segment count 7, got %d", segCount)
	}
}

func TestStreamParser_LargeFile(t *testing.T) {
	// Create a large EDI file with repeated segments
	var buf bytes.Buffer
	buf.WriteString("ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~")
	buf.WriteString("GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~")
	
	// Add 1000 transactions
	for i := 0; i < 1000; i++ {
		buf.WriteString("ST*204*0001~")
		buf.WriteString("B2**SCAC*ABC123*CC~")
		buf.WriteString("S5*1*LD~")
		buf.WriteString("DTM*133*20240102*0800~")
		buf.WriteString("S5*2*UL~")
		buf.WriteString("DTM*132*20240103*1600~")
		buf.WriteString("SE*7*0001~")
	}
	
	buf.WriteString("GE*1000*1~")
	buf.WriteString("IEA*1*000000001~")
	
	reader := bytes.NewReader(buf.Bytes())
	delims := Delimiters{Element: '*', Component: '>', Segment: '~'}
	
	segmentCount := 0
	parser := NewStreamParser(reader, delims, 
		WithBufferSize(1024), // Small buffer to test streaming
		WithSegmentHandler(func(s Segment) error {
			segmentCount++
			return nil
		}))
	
	err := parser.Parse(context.Background())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	expectedSegments := 2 + (7 * 1000) + 2 // ISA/GS + (7 per transaction * 1000) + GE/IEA
	if segmentCount != expectedSegments {
		t.Errorf("Expected %d segments, got %d", expectedSegments, segmentCount)
	}
}

func TestBatchStreamParser(t *testing.T) {
	input := "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~" +
		"GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~" +
		"ST*204*0001~" +
		"B2**SCAC*ABC123*CC~" +
		"S5*1*LD~" +
		"S5*2*UL~" +
		"SE*5*0001~" +
		"GE*1*1~" +
		"IEA*1*000000001~"

	reader := strings.NewReader(input)
	delims := Delimiters{Element: '*', Component: '>', Segment: '~'}
	
	batchCount := 0
	totalSegments := 0
	
	parser := NewBatchStreamParser(reader, delims, 3, func(batch []Segment) error {
		batchCount++
		totalSegments += len(batch)
		
		// Verify batch size (except for last batch)
		if batchCount < 3 && len(batch) != 3 {
			t.Errorf("Batch %d: expected size 3, got %d", batchCount, len(batch))
		}
		
		return nil
	})
	
	err := parser.Parse(context.Background())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if totalSegments != 9 {
		t.Errorf("Expected 9 total segments, got %d", totalSegments)
	}
	
	if batchCount != 3 {
		t.Errorf("Expected 3 batches, got %d", batchCount)
	}
}

func TestStreamParser_ContextCancellation(t *testing.T) {
	// Create a large input that would take time to process
	var buf bytes.Buffer
	for i := 0; i < 10000; i++ {
		buf.WriteString("ST*204*0001~")
	}
	
	reader := bytes.NewReader(buf.Bytes())
	delims := Delimiters{Element: '*', Component: '>', Segment: '~'}
	
	processedCount := 0
	parser := NewStreamParser(reader, delims, WithSegmentHandler(func(s Segment) error {
		processedCount++
		return nil
	}))
	
	// Cancel context after processing some segments
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Let it process a few segments then cancel
		for processedCount < 10 {
			// Wait
		}
		cancel()
	}()
	
	err := parser.Parse(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
	
	// Should have processed some but not all segments
	if processedCount >= 10000 {
		t.Error("Should not have processed all segments")
	}
	if processedCount == 0 {
		t.Error("Should have processed some segments before cancellation")
	}
}

func BenchmarkStreamParser(b *testing.B) {
	// Create test data
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteString("ST*204*0001~B2**SCAC*ABC123*CC~SE*2*0001~")
	}
	data := buf.Bytes()
	
	delims := Delimiters{Element: '*', Component: '>', Segment: '~'}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		parser := NewStreamParser(reader, delims)
		_ = parser.Parse(context.Background())
	}
}

func BenchmarkBatchStreamParser(b *testing.B) {
	// Create test data
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteString("ST*204*0001~B2**SCAC*ABC123*CC~SE*2*0001~")
	}
	data := buf.Bytes()
	
	delims := Delimiters{Element: '*', Component: '>', Segment: '~'}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		parser := NewBatchStreamParser(reader, delims, 10, func(batch []Segment) error {
			return nil
		})
		_ = parser.Parse(context.Background())
	}
}