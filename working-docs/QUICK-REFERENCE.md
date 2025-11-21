# MCP Language Server - Quick Reference Guide

## Essential File Locations (Absolute Paths)

```
/Users/chloelee/Code/mcp-language-server/
├── main.go                              # Main entry point
├── tools.go                             # MCP tool registration
├── internal/
│   ├── tools/
│   │   ├── definition.go                # Definition tool (REFERENCE)
│   │   ├── hover.go                     # Hover tool (SIMPLE EXAMPLE)
│   │   ├── references.go                # References tool (UPDATED PATTERN)
│   │   ├── rename-symbol.go             # Rename tool (COMPLEX)
│   │   ├── diagnostics.go               # Diagnostics tool
│   │   ├── lsp-utilities.go             # LSP helper functions
│   │   ├── utilities.go                 # Text formatting utilities
│   │   └── logging.go                   # Tool logger setup
│   ├── lsp/
│   │   ├── client.go                    # LSP client core
│   │   ├── methods.go                   # LSP methods (AUTO-GENERATED - DO NOT EDIT)
│   │   ├── transport.go                 # LSP JSON-RPC transport
│   │   ├── protocol.go                  # LSP protocol structures
│   │   └── ...
│   ├── protocol/
│   │   ├── tsprotocol.go                # LSP types (AUTO-GENERATED - DO NOT EDIT)
│   │   ├── interfaces.go                # Union type handlers
│   │   └── ...
│   └── utilities/
│       └── edit.go                      # File editing utilities
└── integrationtests/                    # Integration tests & snapshots
```

---

## Core Concepts Quick Lookup

### 1. Index Conventions
| Context | Indexing | Example |
|---------|----------|---------|
| User API | 1-indexed | Line 1, Column 1 = first character |
| LSP Protocol | 0-indexed | Line 0, Column 0 = first character |
| Conversion | `lspValue = userValue - 1` | User line 5 → LSP line 4 |

### 2. Standard Tool Parameter Pattern
```go
// All location-based tools use: filePath, line, column (1-indexed)
GetTool(ctx, client, filePath string, line int, column int) (string, error)

// Symbol-based tools use: symbolName
GetDefinition(ctx, client, symbolName string) (string, error)
```

### 3. File Opening (Required First Step)
```go
err := client.OpenFile(ctx, filePath)
if err != nil {
    return "", fmt.Errorf("could not open file: %v", err)
}
```

### 4. Position Creation
```go
position := protocol.Position{
    Line:      uint32(line - 1),      // Convert to 0-indexed
    Character: uint32(column - 1),    // Convert to 0-indexed
}

uri := protocol.DocumentUri("file://" + filePath)
params := protocol.SomeToolParams{
    TextDocument: protocol.TextDocumentIdentifier{URI: uri},
    Position: position,
}
```

---

## LSP Method Calls

### Existing LSP Client Methods (in methods.go)

```go
// Location-based methods (same signature pattern)
func (c *Client) Definition(ctx context.Context, params protocol.DefinitionParams) 
    (protocol.Or_Result_textDocument_definition, error)

func (c *Client) TypeDefinition(ctx context.Context, params protocol.TypeDefinitionParams) 
    (protocol.Or_Result_textDocument_typeDefinition, error)

func (c *Client) Implementation(ctx context.Context, params protocol.ImplementationParams) 
    (protocol.Or_Result_textDocument_implementation, error)

func (c *Client) Hover(ctx context.Context, params protocol.HoverParams) 
    (protocol.Hover, error)

func (c *Client) References(ctx context.Context, params protocol.ReferenceParams) 
    ([]protocol.Location, error)

// Refactoring methods
func (c *Client) Rename(ctx context.Context, params protocol.RenameParams) 
    (protocol.WorkspaceEdit, error)

// Document info
func (c *Client) DocumentSymbol(ctx context.Context, params protocol.DocumentSymbolParams) 
    (protocol.Or_Result_textDocument_documentSymbol, error)

// Diagnostics
func (c *Client) Diagnostic(ctx context.Context, params protocol.DocumentDiagnosticParams) 
    (protocol.DocumentDiagnosticReport, error)
```

---

## Common Code Patterns

### Pattern 1: Simple Location-Based Tool (like Hover)
```go
package tools

import (
    "context"
    "fmt"
    "github.com/isaacphi/mcp-language-server/internal/lsp"
    "github.com/isaacphi/mcp-language-server/internal/protocol"
)

func GetHoverInfo(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
    // Step 1: Open file
    err := client.OpenFile(ctx, filePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %v", err)
    }

    // Step 2: Create position and params
    position := protocol.Position{
        Line:      uint32(line - 1),
        Character: uint32(column - 1),
    }
    uri := protocol.DocumentUri("file://" + filePath)
    
    params := protocol.HoverParams{
        TextDocument: protocol.TextDocumentIdentifier{URI: uri},
        Position: position,
    }

    // Step 3: Call LSP method
    result, err := client.Hover(ctx, params)
    if err != nil {
        return "", fmt.Errorf("failed to get hover information: %v", err)
    }

    // Step 4: Format and return
    if result.Contents.Value == "" {
        return "No hover information available", nil
    }
    return result.Contents.Value, nil
}
```

### Pattern 2: Tool with Result Collection (like References)
```go
func FindReferences(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
    // Open file
    err := client.OpenFile(ctx, filePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %v", err)
    }

    // Create params
    position := protocol.Position{
        Line:      uint32(line - 1),
        Character: uint32(column - 1),
    }
    uri := protocol.DocumentUri("file://" + filePath)
    
    params := protocol.ReferenceParams{
        TextDocumentPositionParams: protocol.TextDocumentPositionParams{
            TextDocument: protocol.TextDocumentIdentifier{URI: uri},
            Position: position,
        },
        Context: protocol.ReferenceContext{
            IncludeDeclaration: false,
        },
    }

    // Get results
    refs, err := client.References(ctx, params)
    if err != nil {
        return "", fmt.Errorf("failed to get references: %v", err)
    }

    if len(refs) == 0 {
        return "No references found", nil
    }

    // Process and format results
    var output strings.Builder
    for _, ref := range refs {
        output.WriteString(fmt.Sprintf("L%d:C%d\n", 
            ref.Range.Start.Line+1,  // Convert back to 1-indexed for display
            ref.Range.Start.Character+1))
    }
    return output.String(), nil
}
```

### Pattern 3: Tool Registration in tools.go
```go
func (s *mcpServer) registerTools() error {
    coreLogger.Debug("Registering MCP tools")

    // Define tool schema
    myTool := mcp.NewTool("my_tool",
        mcp.WithDescription("Description of what the tool does"),
        mcp.WithString("filePath",
            mcp.Required(),
            mcp.Description("The path to the file..."),
        ),
        mcp.WithNumber("line",
            mcp.Required(),
            mcp.Description("The line number (1-indexed)"),
        ),
        mcp.WithNumber("column",
            mcp.Required(),
            mcp.Description("The column number (1-indexed)"),
        ),
    )

    // Register handler
    s.mcpServer.AddTool(myTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Extract parameters
        filePath, err := request.RequireString("filePath")
        if err != nil {
            return mcp.NewToolResultErrorFromErr("invalid argument", err), nil
        }
        line, err := request.RequireInt("line")
        if err != nil {
            return mcp.NewToolResultErrorFromErr("invalid argument", err), nil
        }
        column, err := request.RequireInt("column")
        if err != nil {
            return mcp.NewToolResultErrorFromErr("invalid argument", err), nil
        }

        // Log execution
        coreLogger.Debug("Executing my_tool for file: %s line: %d column: %d", filePath, line, column)

        // Call business logic
        result, err := tools.MyToolFunction(s.ctx, s.lspClient, filePath, line, column)
        if err != nil {
            coreLogger.Error("Failed to execute my_tool: %v", err)
            return mcp.NewToolResultError(fmt.Sprintf("failed to execute my_tool: %v", err)), nil
        }

        return mcp.NewToolResultText(result), nil
    })

    coreLogger.Info("Successfully registered all MCP tools")
    return nil
}
```

---

## Logging Patterns

### In tools/{tool}.go
```go
import "github.com/isaacphi/mcp-language-server/internal/tools"

// Logger is already defined in tools/logging.go
toolsLogger.Debug("Debug message: %v", value)
toolsLogger.Info("Info message: %v", value)
toolsLogger.Warn("Warning message: %v", err)
toolsLogger.Error("Error message: %v", err)
```

### In main/tools.go (tool registration)
```go
// coreLogger is defined at top of main.go
var coreLogger = logging.NewLogger(logging.Core)

coreLogger.Debug("Registering tools")
coreLogger.Debug("Executing tool for: %s", param)
coreLogger.Error("Failed to execute tool: %v", err)
coreLogger.Info("Successfully registered tools")
```

---

## Utility Functions (Already Available)

### From tools/utilities.go
```go
// Extract text content from a location range
text, err := ExtractTextFromLocation(loc protocol.Location) (string, error)

// Check if a range contains a position
contains := containsPosition(r protocol.Range, p protocol.Position) bool

// Add line numbers to text
numbered := addLineNumbers(text string, startLine int) string

// Convert set of lines to continuous ranges
ranges := ConvertLinesToRanges(linesToShow map[int]bool, totalLines int) []LineRange

// Format content with line ranges
formatted := FormatLinesWithRanges(lines []string, ranges []LineRange) string
```

### From tools/lsp-utilities.go
```go
// Get full code block around a location
code, loc, err := GetFullDefinition(ctx context.Context, client *lsp.Client, loc protocol.Location) (string, protocol.Location, error)

// Determine which lines to display with context
lines, err := GetLineRangesToDisplay(ctx context.Context, client *lsp.Client, locations []protocol.Location, totalLines int, contextLines int) (map[int]bool, error)
```

---

## Protocol Types (Most Common)

```go
// Basic types
protocol.Position {
    Line      uint32
    Character uint32
}

protocol.Range {
    Start protocol.Position
    End   protocol.Position
}

protocol.Location {
    URI   protocol.DocumentUri  // "file:///path/to/file.go"
    Range protocol.Range
}

protocol.DocumentUri string  // "file://" + filepath

// TextDocument identifiers
protocol.TextDocumentIdentifier {
    URI protocol.DocumentUri
}

// Common params (embed TextDocumentPositionParams)
protocol.TextDocumentPositionParams {
    TextDocument protocol.TextDocumentIdentifier
    Position     protocol.Position
}

// Specific param types
protocol.HoverParams {
    TextDocumentPositionParams
}

protocol.ReferenceParams {
    TextDocumentPositionParams
    Context protocol.ReferenceContext {
        IncludeDeclaration bool
    }
}

protocol.RenameParams {
    TextDocument protocol.TextDocumentIdentifier
    Position     protocol.Position
    NewName      string
}

// Result types
protocol.Hover {
    Contents protocol.MarkupContent {
        Kind  string  // "markdown" or "plaintext"
        Value string  // The actual content
    }
}

protocol.WorkspaceEdit {
    Changes         map[protocol.DocumentUri][]protocol.TextEdit
    DocumentChanges []protocol.DocumentChange
}
```

---

## Error Handling Conventions

```go
// Wrap errors with context
if err != nil {
    return "", fmt.Errorf("failed to <operation>: %v", err)
}

// Tool handler error returns
if err != nil {
    coreLogger.Error("Failed to execute tool: %v", err)
    return mcp.NewToolResultError(fmt.Sprintf("failed to execute tool: %v", err)), nil
}

// Parameter extraction errors
if err != nil {
    return mcp.NewToolResultErrorFromErr("invalid argument", err), nil
}
```

---

## File Opening Best Practices

```go
// Always check if file needs opening before using it
err := client.OpenFile(ctx, filePath)
if err != nil {
    // File might already be open, check if it's critical
    if !client.IsFileOpen(filePath) {
        return "", fmt.Errorf("could not open file: %v", err)
    }
}

// When done with operations that modified files
// (like rename), files remain open for future use
```

---

## Tips for New Tool Implementation

1. **Start with a simple tool**: Copy hover.go structure first
2. **Use location-based params** (filePath, line, column) for consistency
3. **Always convert 1-indexed → 0-indexed** for LSP
4. **Always open files first** before using them
5. **Log debug messages** at key points
6. **Wrap errors** with context about what operation failed
7. **Format output** with file headers and line numbers
8. **Group results** by file for readability
9. **Use existing utilities** for text extraction and formatting
10. **Don't edit methods.go or tsprotocol.go** - these are auto-generated

