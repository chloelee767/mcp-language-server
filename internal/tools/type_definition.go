package tools

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func GetTypeDefinition(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
	err := client.OpenFile(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}

	position := protocol.Position{
		Line:      uint32(line - 1),
		Character: uint32(column - 1),
	}
	uri := protocol.DocumentUri("file://" + filePath)

	params := protocol.TypeDefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri,
			},
			Position: position,
		},
	}

	result, err := client.TypeDefinition(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get type definition: %v", err)
	}

	locations, err := ExtractLocationsFromDefinitionResult(result.Value)
	if err != nil {
		return "", fmt.Errorf("failed to parse type definition locations: %v", err)
	}

	if len(locations) == 0 {
		return "No type definition found", nil
	}

	locationsByFile := make(map[protocol.DocumentUri][]protocol.Location)
	for _, loc := range locations {
		locationsByFile[loc.URI] = append(locationsByFile[loc.URI], loc)
	}

	uris := make([]string, 0, len(locationsByFile))
	for uri := range locationsByFile {
		uris = append(uris, string(uri))
	}
	sort.Strings(uris)

	var allDefinitions []string
	for _, uriStr := range uris {
		uri := protocol.DocumentUri(uriStr)
		fileLocs := locationsByFile[uri]
		filePath := strings.TrimPrefix(uriStr, "file://")

		fileInfo := fmt.Sprintf("---\n\nFile: %s\n", filePath)

		var locStrings []string
		for _, loc := range fileLocs {
			locStr := fmt.Sprintf("L%d:C%d",
				loc.Range.Start.Line+1,
				loc.Range.Start.Character+1)
			locStrings = append(locStrings, locStr)
		}

		if len(locStrings) > 0 {
			fileInfo += "At: " + strings.Join(locStrings, ", ") + "\n\n"
		}

		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			allDefinitions = append(allDefinitions, fileInfo+"Error reading file: "+err.Error())
			continue
		}

		lines := strings.Split(string(fileContent), "\n")

		linesToShow, err := GetLineRangesToDisplay(ctx, client, fileLocs, len(lines), 5)
		if err != nil {
			continue
		}

		lineRanges := ConvertLinesToRanges(linesToShow, len(lines))
		formattedOutput := fileInfo + FormatLinesWithRanges(lines, lineRanges)
		allDefinitions = append(allDefinitions, formattedOutput)
	}

	return strings.Join(allDefinitions, "\n"), nil
}
