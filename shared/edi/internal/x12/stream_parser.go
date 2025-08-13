package x12

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
)

// StreamParser provides efficient streaming parsing for large EDI files
type StreamParser struct {
	reader     io.Reader
	delims     Delimiters
	bufferSize int
	onSegment  func(Segment) error
	onError    func(error)
	
	// Metrics
	segmentCount int64
	byteCount    int64
	mu           sync.RWMutex
}

// StreamParserOption configures the stream parser
type StreamParserOption func(*StreamParser)

// WithBufferSize sets the buffer size for streaming
func WithBufferSize(size int) StreamParserOption {
	return func(sp *StreamParser) {
		sp.bufferSize = size
	}
}

// WithSegmentHandler sets the callback for each parsed segment
func WithSegmentHandler(handler func(Segment) error) StreamParserOption {
	return func(sp *StreamParser) {
		sp.onSegment = handler
	}
}

// WithErrorHandler sets the error handling callback
func WithErrorHandler(handler func(error)) StreamParserOption {
	return func(sp *StreamParser) {
		sp.onError = handler
	}
}

// NewStreamParser creates a new streaming parser
func NewStreamParser(r io.Reader, d Delimiters, opts ...StreamParserOption) *StreamParser {
	sp := &StreamParser{
		reader:     r,
		delims:     d,
		bufferSize: 64 * 1024, // 64KB default
		onSegment:  func(s Segment) error { return nil },
		onError:    func(e error) {},
	}
	
	for _, opt := range opts {
		opt(sp)
	}
	
	return sp
}

// Parse streams through the EDI file processing segments
func (sp *StreamParser) Parse(ctx context.Context) error {
	scanner := bufio.NewScanner(sp.reader)
	scanner.Buffer(make([]byte, sp.bufferSize), sp.bufferSize*2)
	
	// Custom split function for EDI segments
	scanner.Split(sp.segmentSplitter)
	
	segmentIndex := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		data := scanner.Bytes()
		sp.updateMetrics(int64(len(data)))
		
		// Parse the segment
		segment, err := sp.parseSegmentData(data, segmentIndex)
		if err != nil {
			sp.onError(fmt.Errorf("segment %d: %w", segmentIndex, err))
			continue
		}
		
		// Process the segment
		if err := sp.onSegment(segment); err != nil {
			return fmt.Errorf("handler error at segment %d: %w", segmentIndex, err)
		}
		
		segmentIndex++
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	
	return nil
}

// segmentSplitter is a custom split function for the scanner
func (sp *StreamParser) segmentSplitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Look for segment terminator
	for i := 0; i < len(data); i++ {
		if data[i] == sp.delims.Segment {
			// Return the segment without the terminator
			return i + 1, data[:i], nil
		}
	}
	
	// If at EOF and we have data, return it as the last segment
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	
	// Request more data
	return 0, nil, nil
}

// parseSegmentData parses a single segment's data
func (sp *StreamParser) parseSegmentData(data []byte, index int) (Segment, error) {
	if len(data) == 0 {
		return Segment{}, fmt.Errorf("empty segment")
	}
	
	// Split by element separator
	elements := splitByByte(data, sp.delims.Element)
	if len(elements) == 0 {
		return Segment{}, fmt.Errorf("no elements in segment")
	}
	
	tag := string(elements[0])
	parsedElements := make([][]string, 0, len(elements)-1)
	
	for _, elem := range elements[1:] {
		if sp.delims.Component != 0 && containsByte(elem, sp.delims.Component) {
			// Split components
			components := splitByByte(elem, sp.delims.Component)
			strComponents := make([]string, len(components))
			for i, comp := range components {
				strComponents[i] = string(comp)
			}
			parsedElements = append(parsedElements, strComponents)
		} else {
			parsedElements = append(parsedElements, []string{string(elem)})
		}
	}
	
	return Segment{
		Tag:      tag,
		Elements: parsedElements,
		Index:    index,
	}, nil
}

// updateMetrics updates parsing metrics
func (sp *StreamParser) updateMetrics(bytes int64) {
	sp.mu.Lock()
	sp.segmentCount++
	sp.byteCount += bytes
	sp.mu.Unlock()
}

// GetMetrics returns current parsing metrics
func (sp *StreamParser) GetMetrics() (segments int64, bytes int64) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.segmentCount, sp.byteCount
}

// splitByByte splits data by a delimiter byte
func splitByByte(data []byte, delim byte) [][]byte {
	var result [][]byte
	start := 0
	
	for i := 0; i < len(data); i++ {
		if data[i] == delim {
			result = append(result, data[start:i])
			start = i + 1
		}
	}
	
	// Add the last segment
	if start < len(data) {
		result = append(result, data[start:])
	} else if start == len(data) {
		// Empty element at the end
		result = append(result, []byte{})
	}
	
	return result
}

// containsByte checks if a byte slice contains a specific byte
func containsByte(data []byte, b byte) bool {
	for _, d := range data {
		if d == b {
			return true
		}
	}
	return false
}

// BatchStreamParser processes EDI in batches for better performance
type BatchStreamParser struct {
	*StreamParser
	batchSize int
	batch     []Segment
	onBatch   func([]Segment) error
}

// NewBatchStreamParser creates a batch processing stream parser
func NewBatchStreamParser(r io.Reader, d Delimiters, batchSize int, onBatch func([]Segment) error) *BatchStreamParser {
	bsp := &BatchStreamParser{
		batchSize: batchSize,
		batch:     make([]Segment, 0, batchSize),
		onBatch:   onBatch,
	}
	
	bsp.StreamParser = NewStreamParser(r, d, WithSegmentHandler(bsp.handleSegment))
	return bsp
}

// handleSegment accumulates segments into batches
func (bsp *BatchStreamParser) handleSegment(seg Segment) error {
	bsp.batch = append(bsp.batch, seg)
	
	if len(bsp.batch) >= bsp.batchSize {
		if err := bsp.processBatch(); err != nil {
			return err
		}
	}
	
	return nil
}

// processBatch processes the accumulated batch
func (bsp *BatchStreamParser) processBatch() error {
	if len(bsp.batch) == 0 {
		return nil
	}
	
	err := bsp.onBatch(bsp.batch)
	bsp.batch = bsp.batch[:0] // Reset batch
	return err
}

// Parse overrides to handle final batch
func (bsp *BatchStreamParser) Parse(ctx context.Context) error {
	if err := bsp.StreamParser.Parse(ctx); err != nil {
		return err
	}
	
	// Process any remaining segments in the batch
	return bsp.processBatch()
}