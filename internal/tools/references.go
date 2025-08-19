package tools

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func FindReferences(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
	// Get context lines from environment variable
	contextLines := 5
	if envLines := os.Getenv("LSP_CONTEXT_LINES"); envLines != "" {
		if val, err := strconv.Atoi(envLines); err == nil && val >= 0 {
			contextLines = val
		}
	}

	// Open the file if not already open
	err := client.OpenFile(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}

	// Convert 1-indexed line/column to 0-indexed for LSP protocol
	position := protocol.Position{
		Line:      uint32(line - 1),
		Character: uint32(column - 1),
	}
	uri := protocol.DocumentUri("file://" + filePath)

	// Use LSP references request with correct params structure
	refsParams := protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri,
			},
			Position: position,
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	}

	refs, err := client.References(ctx, refsParams)
	if err != nil {
		return "", fmt.Errorf("failed to get references: %v", err)
	}

	if len(refs) == 0 {
		return "No references found", nil
	}

	// Group references by file
	refsByFile := make(map[protocol.DocumentUri][]protocol.Location)
	for _, ref := range refs {
		refsByFile[ref.URI] = append(refsByFile[ref.URI], ref)
	}

	// Get sorted list of URIs
	uris := make([]string, 0, len(refsByFile))
	for uri := range refsByFile {
		uris = append(uris, string(uri))
	}
	sort.Strings(uris)

	var allReferences []string
	// Process each file's references in sorted order
	for _, uriStr := range uris {
		uri := protocol.DocumentUri(uriStr)
		fileRefs := refsByFile[uri]
		filePath := strings.TrimPrefix(uriStr, "file://")

		// Format file header
		fileInfo := fmt.Sprintf("---\n\n%s\nReferences in File: %d\n",
			filePath,
			len(fileRefs),
		)

		// Format locations with context
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			// Log error but continue with other files
			allReferences = append(allReferences, fileInfo+"\nError reading file: "+err.Error())
			continue
		}

		lines := strings.Split(string(fileContent), "\n")

		// Track reference locations for header display
		var locStrings []string
		for _, ref := range fileRefs {
			locStr := fmt.Sprintf("L%d:C%d",
				ref.Range.Start.Line+1,
				ref.Range.Start.Character+1)
			locStrings = append(locStrings, locStr)
		}

		// Collect lines to display using the utility function
		linesToShow, err := GetLineRangesToDisplay(ctx, client, fileRefs, len(lines), contextLines)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		// Convert to line ranges using the utility function
		lineRanges := ConvertLinesToRanges(linesToShow, len(lines))

		// Format with locations in header
		formattedOutput := fileInfo
		if len(locStrings) > 0 {
			formattedOutput += "At: " + strings.Join(locStrings, ", ") + "\n"
		}

		// Format the content with ranges
		formattedOutput += "\n" + FormatLinesWithRanges(lines, lineRanges)
		allReferences = append(allReferences, formattedOutput)
	}

	return strings.Join(allReferences, "\n"), nil
}
