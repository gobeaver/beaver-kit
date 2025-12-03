package filekit

import (
	"context"
	"path/filepath"
	"strings"
)

// ============================================================================
// FileSelector Interface
// ============================================================================

// FileSelector defines the interface for filtering files during listing operations.
// Inspired by Apache Commons VFS FileSelector - proven stable for 20+ years.
//
// This interface is designed to be:
// - Future-proof: New selector types can be added without breaking existing code
// - Composable: Selectors can be combined with And, Or, Not
// - Driver-optimizable: Drivers can inspect selectors for native optimizations
//
// Example usage:
//
//	// Simple glob selector
//	files, err := filekit.ListWithSelector(ctx, fs, "/", filekit.Glob("*.txt"))
//
//	// Composed selector
//	selector := filekit.And(
//	    filekit.Glob("*.jpg"),
//	    filekit.FuncSelector(func(f *filekit.FileInfo) bool {
//	        return f.Size < 10*1024*1024
//	    }),
//	)
//	files, err := filekit.ListWithSelector(ctx, fs, "/images", selector)
type FileSelector interface {
	// Match returns true if the file should be included in results.
	Match(file *FileInfo) bool

	// TraverseDescendants returns true if directory descendants should be traversed.
	// This enables early termination for deep directory trees.
	// If false, the directory and all its contents are skipped.
	// Only called for directories (file.IsDir == true).
	TraverseDescendants(file *FileInfo) bool
}

// ============================================================================
// ListWithSelector - Main API (like VFS findFiles)
// ============================================================================

// ListWithSelector lists files matching the given selector.
// Set recursive to true for deep traversal (like VFS findFiles).
//
// Example:
//
//	// List all JPEG files recursively
//	files, err := filekit.ListWithSelector(ctx, fs, "/images", filekit.Glob("*.jpg"), true)
//
//	// List immediate children only
//	files, err := filekit.ListWithSelector(ctx, fs, "/", filekit.All(), false)
func ListWithSelector(ctx context.Context, fs FileSystem, path string, selector FileSelector, recursive bool) ([]FileInfo, error) {
	if selector == nil {
		selector = All()
	}

	var results []FileInfo
	err := listRecursive(ctx, fs, path, selector, recursive, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func listRecursive(ctx context.Context, fs FileSystem, path string, selector FileSelector, recursive bool, results *[]FileInfo) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	files, err := fs.ListContents(ctx, path, false)
	if err != nil {
		return err
	}

	for i := range files {
		file := &files[i]

		if file.IsDir {
			// Recurse if enabled and selector allows
			if recursive && selector.TraverseDescendants(file) {
				if err := listRecursive(ctx, fs, file.Path, selector, recursive, results); err != nil {
					return err
				}
			}
		} else {
			// Regular file - check if matches
			if selector.Match(file) {
				*results = append(*results, *file)
			}
		}
	}

	return nil
}

// ============================================================================
// Built-in Selectors (VFS-equivalent: AllFileSelector, PatternFileSelector, FileDepthSelector)
// ============================================================================

// AllSelector matches all files and traverses all directories.
type AllSelector struct{}

func (s AllSelector) Match(file *FileInfo) bool              { return true }
func (s AllSelector) TraverseDescendants(file *FileInfo) bool { return true }

// All returns a selector that matches all files (like VFS AllFileSelector).
func All() FileSelector {
	return AllSelector{}
}

// ============================================================================
// Glob - Pattern matching (like VFS PatternFileSelector/WildcardFileSelector)
// ============================================================================

type globSelector struct {
	pattern string
}

// Glob creates a selector using glob patterns (like VFS WildcardFileSelector).
// Supports: *, ?, [abc], [a-z]
//
// Examples:
//
//	Glob("*.txt")           // All .txt files
//	Glob("image_????.jpg")  // image_0001.jpg, etc.
//	Glob("[a-z]*.go")       // Go files starting with lowercase
func Glob(pattern string) FileSelector {
	return &globSelector{pattern: pattern}
}

func (s *globSelector) Match(file *FileInfo) bool {
	matched, err := filepath.Match(s.pattern, file.Name)
	if err != nil {
		return false
	}
	return matched
}

func (s *globSelector) TraverseDescendants(file *FileInfo) bool {
	return true
}

// ============================================================================
// Depth - Depth limiting (like VFS FileDepthSelector)
// ============================================================================

type depthSelector struct {
	maxDepth int
	basePath string
}

// Depth limits traversal to maxDepth levels (like VFS FileDepthSelector).
// Depth 1 = immediate children only.
//
// Example:
//
//	Depth(2, "/")  // Up to 2 levels deep from root
func Depth(maxDepth int, basePath string) FileSelector {
	return &depthSelector{
		maxDepth: maxDepth,
		basePath: strings.TrimSuffix(basePath, "/"),
	}
}

func (s *depthSelector) getDepth(path string) int {
	rel := strings.TrimPrefix(path, s.basePath)
	rel = strings.Trim(rel, "/")
	if rel == "" {
		return 0
	}
	return strings.Count(rel, "/") + 1
}

func (s *depthSelector) Match(file *FileInfo) bool {
	return s.getDepth(file.Path) <= s.maxDepth
}

func (s *depthSelector) TraverseDescendants(file *FileInfo) bool {
	return s.getDepth(file.Path) < s.maxDepth
}

// ============================================================================
// Composable Selectors (And, Or, Not)
// ============================================================================

type andSelector struct {
	selectors []FileSelector
}

// And matches only if ALL selectors match.
func And(selectors ...FileSelector) FileSelector {
	return &andSelector{selectors: selectors}
}

func (s *andSelector) Match(file *FileInfo) bool {
	for _, sel := range s.selectors {
		if !sel.Match(file) {
			return false
		}
	}
	return true
}

func (s *andSelector) TraverseDescendants(file *FileInfo) bool {
	for _, sel := range s.selectors {
		if sel.TraverseDescendants(file) {
			return true
		}
	}
	return false
}

type orSelector struct {
	selectors []FileSelector
}

// Or matches if ANY selector matches.
func Or(selectors ...FileSelector) FileSelector {
	return &orSelector{selectors: selectors}
}

func (s *orSelector) Match(file *FileInfo) bool {
	for _, sel := range s.selectors {
		if sel.Match(file) {
			return true
		}
	}
	return false
}

func (s *orSelector) TraverseDescendants(file *FileInfo) bool {
	for _, sel := range s.selectors {
		if sel.TraverseDescendants(file) {
			return true
		}
	}
	return false
}

type notSelector struct {
	selector FileSelector
}

// Not inverts a selector's match result.
func Not(selector FileSelector) FileSelector {
	return &notSelector{selector: selector}
}

func (s *notSelector) Match(file *FileInfo) bool {
	return !s.selector.Match(file)
}

func (s *notSelector) TraverseDescendants(file *FileInfo) bool {
	return true
}

// ============================================================================
// FuncSelector - Custom logic (escape hatch for any use case)
// ============================================================================

type funcSelector struct {
	matchFn    func(*FileInfo) bool
	traverseFn func(*FileInfo) bool
}

// FuncSelector creates a selector from a custom function.
// This is the escape hatch for any filtering logic not covered by built-ins.
//
// Example:
//
//	FuncSelector(func(f *filekit.FileInfo) bool {
//	    return f.Size > 1024 && strings.Contains(f.Name, "report")
//	})
func FuncSelector(fn func(*FileInfo) bool) FileSelector {
	return &funcSelector{
		matchFn:    fn,
		traverseFn: func(*FileInfo) bool { return true },
	}
}

// FuncSelectorFull creates a selector with custom match and traverse functions.
func FuncSelectorFull(matchFn, traverseFn func(*FileInfo) bool) FileSelector {
	return &funcSelector{
		matchFn:    matchFn,
		traverseFn: traverseFn,
	}
}

func (s *funcSelector) Match(file *FileInfo) bool              { return s.matchFn(file) }
func (s *funcSelector) TraverseDescendants(file *FileInfo) bool { return s.traverseFn(file) }
