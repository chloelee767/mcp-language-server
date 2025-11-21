# MCP Language Server Tool Implementation Guide

## Overview
This document provides a comprehensive guide to understanding how MCP server tools are implemented in the `mcp-language-server` codebase and provides a reference for implementing new tools like `typeDefinition` and `implementation`.

---

## 1. Tool Architecture Overview

The MCP server tool implementation follows this flow:

```
User Request
    ↓
tools.go (MCP tool registration & request handling)
    ↓
internal/tools/{tool-name}.go (Business logic)
    ↓
internal/lsp/client.go (LSP Client methods)
    ↓
internal/lsp/methods.go (Generated LSP request methods)
    ↓
internal/lsp/transport.go (Call/Notify methods for LSP communication)
    ↓
Language Server (gopls, rust-analyzer, etc.)
```

---

## 2. File Locations and Responsibilities

### Key Files:

1. **tools.go** (Root level)
   - Location: `/Users/chloelee/Code/mcp-language-server/tools.go`
   - Responsibility: Tool registration and MCP request/response handling
   - Contains: `registerTools()` function that defines all MCP tools

2. **internal/tools/{tool-name}.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/tools/`
   - Responsibility: Tool-specific business logic
   - Examples: `definition.go`, `hover.go`, `references.go`, `rename-symbol.go`

3. **internal/tools/lsp-utilities.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/tools/lsp-utilities.go`
   - Responsibility: Shared utilities for LSP operations
   - Key functions: `GetFullDefinition()`, `GetLineRangesToDisplay()`

4. **internal/tools/utilities.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/tools/utilities.go`
   - Responsibility: Text formatting and manipulation utilities
   - Key functions: `ExtractTextFromLocation()`, `addLineNumbers()`, `FormatLinesWithRanges()`

5. **internal/lsp/methods.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/lsp/methods.go`
   - **IMPORTANT: This file is auto-generated** - Do not edit manually
   - Contains: LSP client method wrappers (e.g., `Definition()`, `TypeDefinition()`, `Implementation()`, `Hover()`, `References()`)

6. **internal/lsp/client.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/lsp/client.go`
   - Responsibility: LSP client initialization and core methods
   - Key method: `Call(ctx, method, params, result)` - Generic LSP request method

7. **internal/lsp/transport.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/lsp/transport.go`
   - Responsibility: LSP protocol message transport
   - Key methods: `Call()`, `Notify()`, `WriteMessage()`, `ReadMessage()`

8. **internal/protocol/tsprotocol.go**
   - Location: `/Users/chloelee/Code/mcp-language-server/internal/protocol/`
   - **IMPORTANT: This file is auto-generated** - Do not edit manually
   - Contains: LSP protocol type definitions (params, results, capabilities)

---

## 3. Request/Response Formats

### Standard Pattern for File + Position Parameters

Modern tools use the pattern: `filePath`, `line`, `column` (1-indexed)

**From tools.go (line 122-162) - References tool example:**
```go
findReferencesTool := mcp.NewTool("references",
    mcp.WithDescription("Find all usages and references of a symbol..."),
    mcp.WithString("filePath",
        mcp.Required(),
        mcp.Description("The path to the file containing the symbol to find references for"),
    ),
    mcp.WithNumber("line",
        mcp.Required(),
        mcp.Description("The line number where the symbol is located (1-indexed)"),
    ),
    mcp.WithNumber("column",
        mcp.Required(),
        mcp.Description("The column number where the symbol is located (1-indexed)"),
    ),
)

s.mcpServer.AddTool(findReferencesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

    coreLogger.Debug("Executing references for file: %s line: %d column: %d", filePath, line, column)
    text, err := tools.FindReferences(s.ctx, s.lspClient, filePath, line, column)
    if err != nil {
        coreLogger.Error("Failed to find references: %v", err)
        return mcp.NewToolResultError(fmt.Sprintf("failed to find references: %v", err)), nil
    }
    return mcp.NewToolResultText(text), nil
})
```

### Other Parameter Patterns

**Symbol-based (Definition tool):**
```go
mcp.WithString("symbolName",
    mcp.Required(),
    mcp.Description("The name of the symbol whose definition you want to find (e.g. 'mypackage.MyFunction', 'MyType.MyMethod')"),
)
```

**Optional boolean parameters (Diagnostics tool):**
```go
mcp.WithBoolean("contextLines",
    mcp.Description("Lines to include around each diagnostic."),
    mcp.DefaultBool(false),
)
```

### LSP Protocol Type Conversions

All tools convert user-facing 1-indexed line/column numbers to 0-indexed for LSP:

```go
// From tools/hover.go (line 20-26)
position := protocol.Position{
    Line:      uint32(line - 1),        // Convert 1-indexed to 0-indexed
    Character: uint32(column - 1),      // Convert 1-indexed to 0-indexed
}
uri := protocol.DocumentUri("file://" + filePath)
params := protocol.HoverParams{}
params.TextDocument = protocol.TextDocumentIdentifier{URI: uri}
params.Position = position
```

---

## 4. LSP Protocol Method Invocation

### How to Call LSP Methods

**Generic Call Method (lowest level):**
```go
// From lsp/transport.go (line 195)
func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
    // ... request ID generation, channel setup ...
    // Sends JSON-RPC request to language server
    // Waits for response and unmarshals into result type
}
```

**Generated LSP Method Wrappers (primary pattern):**
```go
// From lsp/methods.go (auto-generated)
func (c *Client) References(ctx context.Context, params protocol.ReferenceParams) ([]protocol.Location, error) {
    var result []protocol.Location
    err := c.Call(ctx, "textDocument/references", params, &result)
    return result, err
}

func (c *Client) Hover(ctx context.Context, params protocol.HoverParams) (protocol.Hover, error) {
    var result protocol.Hover
    err := c.Call(ctx, "textDocument/hover", params, &result)
    return result, err
}

// The two methods we need for new tools:
func (c *Client) TypeDefinition(ctx context.Context, params protocol.TypeDefinitionParams) (protocol.Or_Result_textDocument_typeDefinition, error) {
    var result protocol.Or_Result_textDocument_typeDefinition
    err := c.Call(ctx, "textDocument/typeDefinition", params, &result)
    return result, err
}

func (c *Client) Implementation(ctx context.Context, params protocol.ImplementationParams) (protocol.Or_Result_textDocument_implementation, error) {
    var result protocol.Or_Result_textDocument_implementation
    err := c.Call(ctx, "textDocument/implementation", params, &result)
    return result, err
}
```

### Usage in Tool Implementation

```go
// From tools/hover.go (line 34)
hoverResult, err := client.Hover(ctx, params)
if err != nil {
    return "", fmt.Errorf("failed to get hover information: %v", err)
}

// From tools/references.go (line 50)
refs, err := client.References(ctx, refsParams)
if err != nil {
    return "", fmt.Errorf("failed to get references: %v", err)
}

// From tools/rename-symbol.go (line 43)
workspaceEdit, err := client.Rename(ctx, params)
if err != nil {
    return "", fmt.Errorf("failed to rename symbol: %v", err)
}
```

---

## 5. Protocol Type Mappings

### TextDocumentPositionParams

Many LSP methods use `TextDocumentPositionParams` as base, embedded in more specific types:

```go
// From protocol/tsprotocol.go
type TextDocumentPositionParams struct {
    TextDocument protocol.TextDocumentIdentifier
    Position     protocol.Position
}

// Used by multiple methods:
type HoverParams struct {
    TextDocumentPositionParams
}

type ReferenceParams struct {
    TextDocumentPositionParams
    Context protocol.ReferenceContext
}

type TypeDefinitionParams struct {
    TextDocumentPositionParams
}

type ImplementationParams struct {
    TextDocumentPositionParams
}
```

### Result Types

Different methods return different result types:

```go
// Simple location list
func (c *Client) References(...) ([]protocol.Location, error)

// Union type (can be single location or list)
func (c *Client) Definition(...) (protocol.Or_Result_textDocument_definition, error)

// Hover content
func (c *Client) Hover(...) (protocol.Hover, error)

// Union type for type definition
func (c *Client) TypeDefinition(...) (protocol.Or_Result_textDocument_typeDefinition, error)

// Union type for implementation
func (c *Client) Implementation(...) (protocol.Or_Result_textDocument_implementation, error)

// Workspace edit for refactoring
func (c *Client) Rename(...) (protocol.WorkspaceEdit, error)
```

---

## 6. Existing Tool Examples

### Simple Tool: Hover (tools/hover.go)

```go
func GetHoverInfo(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
    // Open file
    err := client.OpenFile(ctx, filePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %v", err)
    }

    // Convert 1-indexed to 0-indexed
    position := protocol.Position{
        Line:      uint32(line - 1),
        Character: uint32(column - 1),
    }
    uri := protocol.DocumentUri("file://" + filePath)
    
    // Create params
    params := protocol.HoverParams{
        TextDocument: protocol.TextDocumentIdentifier{URI: uri},
        Position: position,
    }

    // Call LSP method
    hoverResult, err := client.Hover(ctx, params)
    if err != nil {
        return "", fmt.Errorf("failed to get hover information: %v", err)
    }

    // Format result
    var result strings.Builder
    if hoverResult.Contents.Value == "" {
        // Handle empty result
        result.WriteString("No hover information available...")
    } else {
        result.WriteString(hoverResult.Contents.Value)
    }

    return result.String(), nil
}
```

### Complex Tool: Rename Symbol (tools/rename-symbol.go)

```go
func RenameSymbol(ctx context.Context, client *lsp.Client, filePath string, line, column int, newName string) (string, error) {
    // Open file
    err := client.OpenFile(ctx, filePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %v", err)
    }

    // Convert 1-indexed to 0-indexed
    uri := protocol.DocumentUri("file://" + filePath)
    position := protocol.Position{
        Line:      uint32(line - 1),
        Character: uint32(column - 1),
    }

    // Create params
    params := protocol.RenameParams{
        TextDocument: protocol.TextDocumentIdentifier{URI: uri},
        Position: position,
        NewName:  newName,
    }

    // Call LSP method
    workspaceEdit, err := client.Rename(ctx, params)
    if err != nil {
        return "", fmt.Errorf("failed to rename symbol: %v", err)
    }

    // Process and apply results
    changeCount := 0
    fileCount := 0
    
    // Count changes in Changes field
    if workspaceEdit.Changes != nil {
        fileCount = len(workspaceEdit.Changes)
        for uri, edits := range workspaceEdit.Changes {
            changeCount += len(edits)
            // ... format output ...
        }
    }

    // Apply the workspace edit to files
    if err := utilities.ApplyWorkspaceEdit(workspaceEdit); err != nil {
        return "", fmt.Errorf("failed to apply changes: %v", err)
    }

    return fmt.Sprintf("Successfully renamed symbol to '%s'.\nUpdated %d occurrences across %d files:\n%s",
        newName, changeCount, fileCount, locationsBuilder.String()), nil
}
```

### Intermediate Tool: References (tools/references.go)

```go
func FindReferences(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
    // Open file
    err := client.OpenFile(ctx, filePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %v", err)
    }

    // Convert 1-indexed to 0-indexed
    position := protocol.Position{
        Line:      uint32(line - 1),
        Character: uint32(column - 1),
    }
    uri := protocol.DocumentUri("file://" + filePath)

    // Create params with ReferenceContext
    refsParams := protocol.ReferenceParams{
        TextDocumentPositionParams: protocol.TextDocumentPositionParams{
            TextDocument: protocol.TextDocumentIdentifier{URI: uri},
            Position: position,
        },
        Context: protocol.ReferenceContext{
            IncludeDeclaration: false,
        },
    }

    // Call LSP method
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

    // Format and return results
    // ... build output string ...
    return output, nil
}
```

---

## 7. Patterns and Conventions

### File Opening Convention
All tools that work with source locations must open the file first:
```go
err := client.OpenFile(ctx, filePath)
if err != nil {
    return "", fmt.Errorf("could not open file: %v", err)
}
```

### Line/Column Index Convention
- **User-facing API**: 1-indexed (line 1, column 1 = first character)
- **LSP Protocol**: 0-indexed (line 0, column 0 = first character)
- **Conversion**: `lspValue = userValue - 1`

### Error Handling Convention
```go
// Wrap errors with context
if err != nil {
    return "", fmt.Errorf("failed to <operation>: %v", err)
}

// Use toolsLogger for logging
toolsLogger.Error("Error getting definition: %v", err)
toolsLogger.Debug("Found symbol: %s", symbol.GetName())
toolsLogger.Warn("failed to extract line at position: %v", err)
```

### Logging Convention
```go
// From tools/logging.go
var toolsLogger = logging.NewLogger(logging.Tools)

// In main/tools.go
coreLogger.Debug("Executing definition for symbol: %s", symbolName)
coreLogger.Error("Failed to get definition: %v", err)
```

### Output Formatting Convention
Tools typically format results with:
- File headers with metadata (count, locations)
- Line numbers for code blocks
- Context lines when relevant
- Sorted/organized output for consistency

Example from references (tools/references.go):
```
---

/path/to/file.go
References in File: 3
At: L15:C5, L20:C10, L35:C8

   12|    x := 1
   13|    if x > 0 {
   14|        fmt.Println(x)  <-- reference
   15|    }
...
```

---

## 8. MCP Tool Registration Pattern

All tools follow the same registration pattern in `tools.go`:

```go
func (s *mcpServer) registerTools() error {
    coreLogger.Debug("Registering MCP tools")

    // 1. Define tool with schema
    toolDef := mcp.NewTool("tool_name",
        mcp.WithDescription("User-facing description..."),
        mcp.WithString("paramName",
            mcp.Required(),
            mcp.Description("Parameter description"),
        ),
        // ... more parameters ...
    )

    // 2. Register handler
    s.mcpServer.AddTool(toolDef, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // 3. Extract parameters with error handling
        param, err := request.RequireString("paramName")
        if err != nil {
            return mcp.NewToolResultErrorFromErr("invalid argument", err), nil
        }

        // 4. Log execution
        coreLogger.Debug("Executing tool_name for param: %s", param)

        // 5. Call business logic function
        result, err := tools.ToolFunction(s.ctx, s.lspClient, param)
        if err != nil {
            coreLogger.Error("Failed to execute tool: %v", err)
            return mcp.NewToolResultError(fmt.Sprintf("failed to execute tool: %v", err)), nil
        }

        // 6. Return result
        return mcp.NewToolResultText(result), nil
    })

    coreLogger.Info("Successfully registered all MCP tools")
    return nil
}
```

---

## 9. Important Design Notes

### Auto-Generated Files
- **DO NOT EDIT**: `internal/lsp/methods.go`
- **DO NOT EDIT**: `internal/protocol/tsprotocol.go`
- These are generated by the code generation system (see README for `just generate`)

### LSP Client Methods Already Available
The following LSP client methods already exist in `methods.go` and are ready to use:

**Already available:**
- `Definition()`
- `TypeDefinition()` ← Already exists!
- `Implementation()` ← Already exists!
- `Hover()`
- `References()`
- `Rename()`
- `PrepareRename()`
- `DocumentSymbol()`
- `Diagnostic()`

### Return Type Handling for Union Types

LSP often returns union types that can be different shapes. The codebase handles this in `internal/protocol/interfaces.go`:

```go
// Union type that can be Location, Location[], or LocationLink[]
type Or_Result_textDocument_definition struct {
    Value interface{}
}

// Implement the interface to extract the actual value
func (t Or_Result_textDocument_definition) Results() ([]LocationLike, error) {
    // ... type assertion and conversion logic ...
}
```

---

## 10. For Adding TypeDefinition and Implementation Tools

### LSP Client Methods
Both methods already exist in `internal/lsp/methods.go`:

```go
func (c *Client) TypeDefinition(ctx context.Context, params protocol.TypeDefinitionParams) 
    (protocol.Or_Result_textDocument_typeDefinition, error)

func (c *Client) Implementation(ctx context.Context, params protocol.ImplementationParams) 
    (protocol.Or_Result_textDocument_implementation, error)
```

### Protocol Types Already Available
- `protocol.TypeDefinitionParams` - Similar to DefinitionParams
- `protocol.ImplementationParams` - Similar to DefinitionParams
- `protocol.Or_Result_textDocument_typeDefinition` - Result type
- `protocol.Or_Result_textDocument_implementation` - Result type

### Implementation Steps
1. Create `internal/tools/type_definition.go` with function signature:
   ```go
   func GetTypeDefinition(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error)
   ```

2. Create `internal/tools/implementation.go` with function signature:
   ```go
   func GetImplementation(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error)
   ```

3. Register both tools in `tools.go` following the standard pattern with filePath, line, column parameters

4. Both tools can follow the pattern used by Definition tool for formatting output

---

## 11. Testing Patterns

The project includes integration tests with snapshot testing:

Location: `integrationtests/tests/{language}/{tool_name}/`

Example test structure (from references):
```
integrationtests/
├── tests/
│   ├── go/
│   │   ├── references/
│   │   │   └── references_test.go
│   └── python/
│       └── references/
│           └── references_test.go
├── snapshots/
│   └── {language}_{tool_name}.golden
├── workspaces/
│   ├── go/
│   │   ├── consumer.go
│   │   └── helper.go
│   └── python/
│       └── test_file.py
```

Tests use snapshot testing:
```bash
go test ./integrationtests/...
UPDATE_SNAPSHOTS=true go test ./integrationtests/...  # Update snapshots
```

---

## Summary Table

| Aspect | Details |
|--------|---------|
| **Tool Definition** | `tools.go` - MCP tool schema and request handler |
| **Business Logic** | `internal/tools/{name}.go` - Implementation |
| **LSP Call** | `internal/lsp/methods.go` (auto-generated) - Client methods |
| **Transport** | `internal/lsp/transport.go` - `Call()` method for JSON-RPC |
| **Protocol Types** | `internal/protocol/tsprotocol.go` (auto-generated) - Params/Results |
| **Logging** | `tools/logging.go` - `toolsLogger` for tool level, `coreLogger` for main |
| **Utilities** | `tools/lsp-utilities.go`, `tools/utilities.go` - Shared helpers |
| **User Index** | 1-indexed (line 1, column 1 = first character) |
| **LSP Index** | 0-indexed (line 0, column 0 = first character) |
| **Response Format** | Plain text with formatted code blocks and line numbers |

