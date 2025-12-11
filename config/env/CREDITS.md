# Credits

## Source
- Package: github.com/caarlos0/env
- Version: v11.3.1
- License: MIT
- Original Author: Carlos Alexandro Becker

## License

The MIT License (MIT)

Copyright (c) 2015-2024 Carlos Alexandro Becker

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Update History

| Date       | Version | Reviewer    | Notes                |
|------------|---------|-------------|----------------------|
| 2025-12-10 | v11.3.1 | @engineer   | Initial vendor       |

## Upstream Changes Reviewed

- v11.3.1: Bug fix for Options.Environment behavior (prevents merging with default environment)
- Retraction of v11.3.0 due to the above bug

## Vendored Files

The following files were vendored from the upstream repository:

- `env.go` - Main package implementation
- `error.go` - Error types and constructors
- `env_tomap.go` - Non-Windows toMap implementation
- `env_tomap_windows.go` - Windows-specific toMap implementation

Test files were intentionally excluded from vendoring.
