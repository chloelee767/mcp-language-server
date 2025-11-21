# MCP Language Server Implementation Documentation

This directory contains comprehensive documentation for understanding and implementing tools in the MCP Language Server codebase.

## Quick Navigation

### Start Here
- **[QUICK-REFERENCE.md](QUICK-REFERENCE.md)** (13KB) - Fast lookup for common patterns, code snippets, and file locations

### Comprehensive Reference
- **[IMPLEMENTATION-GUIDE.md](IMPLEMENTATION-GUIDE.md)** (21KB) - Full architectural guide with detailed explanations

## Document Overview

### QUICK-REFERENCE.md
Perfect for quick lookups while coding. Contains:
- Essential file locations (absolute paths)
- Index conventions and parameter patterns
- Common code patterns (copy-paste ready)
- LSP method signatures
- Logging patterns
- Utility functions
- Protocol types reference
- Error handling conventions
- 10 tips for new tool implementation

**Best for:** Implementing new tools, quick syntax lookups, finding file locations

### IMPLEMENTATION-GUIDE.md
Comprehensive guide for understanding the architecture. Contains:
- Tool architecture overview with flow diagram
- Detailed file responsibilities (8 key files)
- Request/response format conventions
- LSP protocol method invocation (generic and generated methods)
- Protocol type mappings
- 4 existing tool examples (Hover, References, Definition, Rename)
- Patterns and conventions (opening files, indexing, logging, output)
- MCP tool registration pattern
- Design notes on auto-generated files
- Steps for adding TypeDefinition and Implementation tools
- Testing patterns

**Best for:** Understanding the architecture, learning from examples, getting the full picture

## Key Findings at a Glance

### Architecture
```
User Request → tools.go → internal/tools/{tool}.go → LSP Client → LSP Server
```

### Critical Files
- `tools.go` - Tool registration
- `internal/tools/{name}.go` - Business logic
- `internal/lsp/methods.go` - LSP client methods (auto-generated)
- `internal/lsp/transport.go` - JSON-RPC transport
- `internal/protocol/tsprotocol.go` - LSP types (auto-generated)

### For Adding TypeDefinition and Implementation Tools

**Good News:** The LSP client methods already exist!

```go
func (c *Client) TypeDefinition(ctx context.Context, params protocol.TypeDefinitionParams) 
    (protocol.Or_Result_textDocument_typeDefinition, error)

func (c *Client) Implementation(ctx context.Context, params protocol.ImplementationParams) 
    (protocol.Or_Result_textDocument_implementation, error)
```

You only need to:
1. Create `internal/tools/type_definition.go`
2. Create `internal/tools/implementation.go`
3. Register both tools in `tools.go`

Both tools follow the same pattern as existing location-based tools (e.g., Hover).

### Standard Tool Parameters
```
filePath: string (1-indexed)
line: number (1-indexed)
column: number (1-indexed)
```

LSP protocol converts to 0-indexed internally: `lspValue = userValue - 1`

### Logging
- Tools: `var toolsLogger = logging.NewLogger(logging.Tools)`
- Main: `var coreLogger = logging.NewLogger(logging.Core)`

## File Locations (Absolute Paths)

```
/Users/chloelee/Code/mcp-language-server/
├── tools.go                           # MCP tool registration
├── main.go                            # Main entry point
├── internal/
│   ├── tools/
│   │   ├── definition.go              # Example: Symbol-based tool
│   │   ├── hover.go                   # Example: Simple location-based tool
│   │   ├── references.go              # Example: Multiple results
│   │   ├── rename-symbol.go           # Example: Complex with workspace edits
│   │   ├── lsp-utilities.go           # Shared LSP helpers
│   │   ├── utilities.go               # Text formatting utilities
│   │   └── logging.go                 # Tool logger setup
│   ├── lsp/
│   │   ├── methods.go                 # LSP methods (DO NOT EDIT - auto-generated)
│   │   ├── client.go                  # LSP client core
│   │   └── transport.go               # JSON-RPC transport
│   └── protocol/
│       └── tsprotocol.go              # LSP types (DO NOT EDIT - auto-generated)
└── working-docs/                      # This directory
```

## Do Not Edit

These files are auto-generated. Regenerate with `just generate`:
- `internal/lsp/methods.go`
- `internal/protocol/tsprotocol.go`

## Existing Tool Examples (by complexity)

1. **Hover** (tools/hover.go) - SIMPLEST: Single LSP call → formatted text
2. **References** (tools/references.go) - INTERMEDIATE: Multiple locations with grouping
3. **Definition** (tools/definition.go) - COMPLEX: Symbol lookup with code block extraction
4. **Rename** (tools/rename-symbol.go) - MOST COMPLEX: Multi-file refactoring with workspace edits

## Common Utilities Available

From `internal/tools/utilities.go`:
- `ExtractTextFromLocation(loc)` - Get text from range
- `addLineNumbers(text, startLine)` - Format with line numbers
- `FormatLinesWithRanges(lines, ranges)` - Format code blocks
- `ConvertLinesToRanges(linesToShow, totalLines)` - Convert set to ranges

From `internal/tools/lsp-utilities.go`:
- `GetFullDefinition(ctx, client, loc)` - Get full code block around location
- `GetLineRangesToDisplay(ctx, client, locations, totalLines, contextLines)` - Determine context

## Testing

Integration tests with snapshot testing in `integrationtests/`:
```bash
go test ./integrationtests/...  # Run tests
UPDATE_SNAPSHOTS=true go test ./integrationtests/...  # Update snapshots
```

Supported languages: Go, Python, Rust, TypeScript, C/C++ (clangd)

## Next Steps

1. **For understanding the architecture:** Read IMPLEMENTATION-GUIDE.md sections 1-5
2. **For implementing TypeDefinition tool:** Read QUICK-REFERENCE.md "Pattern 1" and IMPLEMENTATION-GUIDE.md section 6 (Hover example)
3. **For implementing Implementation tool:** Same as TypeDefinition
4. **For reference:** IMPLEMENTATION-GUIDE.md section 10 has step-by-step guide

## Quick Code Template

```go
// internal/tools/type_definition.go
package tools

import (
    "context"
    "fmt"
    "github.com/isaacphi/mcp-language-server/internal/lsp"
    "github.com/isaacphi/mcp-language-server/internal/protocol"
)

func GetTypeDefinition(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
    // Open file
    err := client.OpenFile(ctx, filePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %v", err)
    }

    // Create params (1-indexed → 0-indexed)
    position := protocol.Position{
        Line:      uint32(line - 1),
        Character: uint32(column - 1),
    }
    uri := protocol.DocumentUri("file://" + filePath)
    params := protocol.TypeDefinitionParams{
        TextDocumentPositionParams: protocol.TextDocumentPositionParams{
            TextDocument: protocol.TextDocumentIdentifier{URI: uri},
            Position: position,
        },
    }

    // Call LSP
    result, err := client.TypeDefinition(ctx, params)
    if err != nil {
        return "", fmt.Errorf("failed to get type definition: %v", err)
    }

    // Format and return
    locations, err := result.Results()
    if err != nil {
        return "No type definition found", nil
    }

    // Build output string with locations
    var output strings.Builder
    for _, loc := range locations {
        output.WriteString(fmt.Sprintf("L%d:C%d: %s\n", 
            loc.Range.Start.Line+1,
            loc.Range.Start.Character+1,
            loc.URI))
    }
    return output.String(), nil
}
```

## Questions or Issues?

Refer to the comprehensive IMPLEMENTATION-GUIDE.md which contains:
- Complete architectural explanations
- Full code examples from the codebase
- Detailed protocol type mappings
- Design patterns and conventions
