package core

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
)

// StackFrame represents a single frame in the runtime call stack.
type StackFrame struct {
	File     string
	Line     int
	Function string
}

// PanicDetail encapsulates all debug info gathered when a panic occurs.
type PanicDetail struct {
	Message     string
	File        string
	Line        int
	Function    string
	CodeContext string
	Trace       []StackFrame
	Env         map[string]map[string]string
	SearchLinks map[string]string
}

// NewPanicDetail analyzes the recover object and request to assemble debug information.
func NewPanicDetail(err interface{}, r *http.Request) PanicDetail {
	message := fmt.Sprintf("%v", err)

	// Gather call stack frames
	pcs := make([]uintptr, 64)
	n := runtime.Callers(3, pcs) // Skip runtime.Callers, NewPanicDetail and recovery block
	frames := runtime.CallersFrames(pcs[:n])

	var trace []StackFrame
	var offendingFrame *StackFrame

	for {
		frame, more := frames.Next()
		isRuntime := strings.Contains(frame.File, "runtime/") || strings.Contains(frame.Function, "runtime.")

		sf := StackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		}
		trace = append(trace, sf)

		if offendingFrame == nil && !isRuntime && frame.File != "" {
			temp := sf
			offendingFrame = &temp
		}

		if !more {
			break
		}
	}

	detail := PanicDetail{
		Message: message,
		Trace:   trace,
	}

	if offendingFrame != nil {
		detail.File = offendingFrame.File
		detail.Line = offendingFrame.Line
		detail.Function = offendingFrame.Function
		detail.CodeContext = getSourceContext(offendingFrame.File, offendingFrame.Line)
	}

	// Search links — BUG-09: use url.QueryEscape for RFC-3986 compliant encoding
	query := fmt.Sprintf("Go %s", message)
	detail.SearchLinks = map[string]string{
		"google":        "https://www.google.com/search?q=" + url.QueryEscape(query),
		"stackoverflow": "https://stackoverflow.com/search?q=" + url.QueryEscape(query),
	}

	// Environment parameters
	detail.Env = getRequestEnv(r)

	return detail
}

func getSourceContext(file string, targetLine int) string {
	if file == "" || targetLine <= 0 {
		return ""
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(content), "\n")

	start := targetLine - 5
	if start < 0 {
		start = 0
	}
	end := targetLine + 5
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		lineNum := i + 1
		lineText := strings.TrimRight(lines[i], "\r\n")
		marker := "     "
		if lineNum == targetLine {
			marker = " >>> "
		}
		sb.WriteString(fmt.Sprintf("%s%4d: %s\n", marker, lineNum, lineText))
	}
	return sb.String()
}

func getRequestEnv(r *http.Request) map[string]map[string]string {
	env := make(map[string]map[string]string)
	if r == nil {
		return env
	}

	// Query params
	queries := make(map[string]string)
	for k, v := range r.URL.Query() {
		queries[k] = strings.Join(v, ", ")
	}
	if len(queries) > 0 {
		env["Query Parameters"] = maskSensitive(queries)
	}

	// POST Form params
	_ = r.ParseForm()
	form := make(map[string]string)
	for k, v := range r.Form {
		form[k] = strings.Join(v, ", ")
	}
	if len(form) > 0 {
		env["Post Parameters"] = maskSensitive(form)
	}

	// Headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	if len(headers) > 0 {
		env["HTTP Headers"] = maskSensitive(headers)
	}

	return env
}

func maskSensitive(data map[string]string) map[string]string {
	masked := make(map[string]string)
	for k, v := range data {
		lowerK := strings.ToLower(k)
		if strings.Contains(lowerK, "pass") ||
			strings.Contains(lowerK, "token") ||
			strings.Contains(lowerK, "key") ||
			strings.Contains(lowerK, "secret") ||
			strings.Contains(lowerK, "auth") {
			masked[k] = "******** (Masked)"
		} else {
			masked[k] = v
		}
	}
	return masked
}

// RenderIntelligentError renders the premium debugging screen.
func RenderIntelligentError(w http.ResponseWriter, detail PanicDetail) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	tplText := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>NovaFlow Debugger 🚨</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&family=JetBrains+Mono&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0f19;
            --card-bg: #151d30;
            --text-color: #f3f4f6;
            --primary: #ef4444;
            --accent: #f59e0b;
            --border-color: #1e293b;
        }
        body {
            background-color: var(--bg-color);
            color: var(--text-color);
            font-family: 'Inter', sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 1100px;
            margin: 0 auto;
        }
        header {
            background: linear-gradient(135deg, #ef4444, #b91c1c);
            padding: 24px;
            border-radius: 12px;
            margin-bottom: 24px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        header h1 {
            margin: 0;
            font-size: 26px;
            font-weight: 700;
            color: white;
            text-shadow: 0 2px 4px rgba(0,0,0,0.2);
        }
        .search-pills a {
            text-decoration: none;
            color: white;
            padding: 8px 16px;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 600;
            margin-left: 8px;
            transition: opacity 0.2s;
        }
        .search-pills a:hover {
            opacity: 0.9;
        }
        .pill-google { background-color: #4285f4; }
        .pill-so { background-color: #f48024; }
        
        .card {
            background-color: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 24px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }
        .error-message {
            font-size: 18px;
            font-family: 'JetBrains Mono', monospace;
            background-color: rgba(239, 68, 68, 0.1);
            border-left: 5px solid var(--primary);
            padding: 16px;
            border-radius: 0 8px 8px 0;
            color: #fca5a5;
            word-break: break-word;
            margin-bottom: 16px;
        }
        .error-meta {
            font-size: 14px;
            color: #9ca3af;
        }
        .error-meta strong {
            color: var(--text-color);
        }
        
        .section-title {
            font-size: 18px;
            font-weight: 600;
            margin-top: 0;
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            color: #a7f3d0;
        }
        
        pre {
            font-family: 'JetBrains Mono', monospace;
            font-size: 14px;
            background-color: #070a13;
            padding: 20px;
            border-radius: 8px;
            overflow-x: auto;
            border: 1px solid #1e293b;
            margin: 0;
        }
        .code-highlight {
            color: #fca5a5;
            background-color: rgba(239, 68, 68, 0.15);
            font-weight: 600;
            display: inline-block;
            width: 100%;
        }
        
        details {
            background-color: #0f172a;
            border: 1px solid var(--border-color);
            border-radius: 8px;
            margin-bottom: 12px;
            overflow: hidden;
        }
        summary {
            padding: 14px 20px;
            font-weight: 600;
            cursor: pointer;
            outline: none;
            user-select: none;
            transition: background-color 0.2s;
        }
        summary:hover {
            background-color: #1e293b;
        }
        .details-content {
            padding: 20px;
            border-top: 1px solid var(--border-color);
            background-color: #070a13;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
        }
        td {
            padding: 8px 12px;
            border-bottom: 1px solid #1e293b;
            vertical-align: top;
        }
        td.key {
            color: #9ca3af;
            width: 30%;
        }
        td.val {
            color: #6ee7b7;
            word-break: break-all;
        }
        
        .trace-item {
            padding: 10px 0;
            border-bottom: 1px solid #1e293b;
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
        }
        .trace-item:last-child {
            border-bottom: none;
        }
        .trace-fn { color: #f59e0b; }
        .trace-loc { color: #9ca3af; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🚨 NovaFlow Error Recovery</h1>
            <div class="search-pills">
                <a href="{{.SearchLinks.google}}" target="_blank" class="pill-google">🔍 Google</a>
                <a href="{{.SearchLinks.stackoverflow}}" target="_blank" class="pill-so">🥞 StackOverflow</a>
            </div>
        </header>

        <div class="card">
            <div class="error-message">{{.Message}}</div>
            <div class="error-meta">
                {{if .File}}
                Panic occurred in: <strong>{{.Function}}()</strong><br>
                Location: <strong>{{.File}}</strong> (Line <strong>{{.Line}}</strong>)
                {{else}}
                Location could not be traced inside application workspace.
                {{end}}
            </div>
        </div>

        {{if .CodeContext}}
        <div class="card">
            <div class="section-title">💡 Source Code Context</div>
            <pre>{{.CodeContext}}</pre>
        </div>
        {{end}}

        {{if .Trace}}
        <div class="card">
            <div class="section-title">🔍 Stack Trace</div>
            <div>
                {{range .Trace}}
                <div class="trace-item">
                    <span class="trace-fn">{{.Function}}()</span><br>
                    <span class="trace-loc">At {{.File}}:{{.Line}}</span>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        {{if .Env}}
        <div class="card">
            <div class="section-title">🌍 Environment Snapshot</div>
            {{range $title, $data := .Env}}
            <details>
                <summary>{{$title}} ({{len $data}})</summary>
                <div class="details-content">
                    <table>
                        {{range $key, $val := $data}}
                        <tr>
                            <td class="key">{{$key}}</td>
                            <td class="val">{{$val}}</td>
                        </tr>
                        {{end}}
                    </table>
                </div>
            </details>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`

	t, err := template.New("error").Parse(tplText)
	if err != nil {
		_, _ = w.Write([]byte("<h1>Internal Server Error</h1><p>"+detail.Message+"</p>"))
		return
	}
	_ = t.Execute(w, detail)
}
