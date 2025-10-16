## 📋 Pull Request Description

### 🎯 What does this PR do?
<!-- Provide a clear and concise description of what this PR accomplishes -->

### 🔗 Related Issues
<!-- Link to related issues using keywords: Fixes #123, Closes #456, Related to #789 -->
- Fixes #
- Closes #  
- Related to #

### 📊 Type of Change
<!-- Mark the type of change with an [x] -->
- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📚 Documentation update (improvements or additions to documentation)
- [ ] 🎨 Style/formatting changes (no functional changes)
- [ ] ♻️ Refactoring (no functional changes, code improvement)
- [ ] ⚡ Performance improvement
- [ ] 🧪 Test updates (adding or updating tests)
- [ ] 🔧 Build/CI changes (changes to build process or CI configuration)

## 🧪 Testing

### ✅ Test Coverage
<!-- Describe how this change has been tested -->
- [ ] Unit tests pass (`go test ./...`)
- [ ] Integration tests pass (`go test ./test/integration/...`)
- [ ] Benchmark tests pass (`go test -bench=. ./test/benchmark/...`)
- [ ] Manual testing completed
- [ ] Added new tests for this change

### 🔍 Test Details
<!-- Provide details about testing performed -->
```bash
# Commands used to test this change
go test ./pkg/...
go test -race ./...
go test -bench=. ./...
```

**Test Environment:**
- OS: [e.g. Ubuntu 20.04, Windows 11, macOS 12]
- Go Version: [e.g. 1.21.0]
- Redis Version: [e.g. 7.0.5]

### 📊 Performance Impact
<!-- If applicable, describe performance implications -->
- [ ] No performance impact
- [ ] Performance improvement (provide benchmarks)
- [ ] Performance regression (explain why acceptable)
- [ ] Performance impact unknown/needs investigation

## 📝 Changes Made

### 🔄 Code Changes
<!-- List the main changes made to the codebase -->
- 
- 
- 

### 🗂️ Files Modified
<!-- List key files that were modified -->
- `pkg/`: 
- `cmd/`: 
- `docs/`: 
- `test/`: 

### ⚙️ Configuration Changes
<!-- List any configuration or environment variable changes -->
- [ ] Added new environment variables
- [ ] Modified existing configuration options
- [ ] Updated default values
- [ ] Added new CLI flags

Details:
```yaml
# New or modified configuration
```

## 📖 Documentation

### 📚 Documentation Updates
- [ ] Updated code comments
- [ ] Updated README.md
- [ ] Updated API documentation  
- [ ] Added/updated examples
- [ ] Updated configuration docs
- [ ] Added troubleshooting guide

### 📋 Breaking Changes
<!-- If this is a breaking change, document the migration path -->
**Migration Guide:**
```go
// Old way
oldMethod()

// New way  
newMethod()
```

## 🔍 Review Guidelines

### 👀 Review Focus Areas
<!-- Help reviewers focus on specific aspects -->
- [ ] Code quality and style
- [ ] Performance implications
- [ ] Security considerations
- [ ] API design
- [ ] Error handling
- [ ] Test coverage
- [ ] Documentation completeness

### 🚨 Special Considerations
<!-- Highlight anything reviewers should pay special attention to -->
- 
- 

## 📷 Screenshots/Demos
<!-- If applicable, add screenshots, GIFs, or demo links -->

### Before
<!-- Screenshots or description of behavior before changes -->

### After  
<!-- Screenshots or description of behavior after changes -->

## ✅ Checklist

### 🧑‍💻 Development
- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published

### 📋 Documentation
- [ ] I have updated relevant documentation
- [ ] I have added examples for new features
- [ ] I have updated configuration documentation if needed
- [ ] Breaking changes are documented with migration guide

### 🤝 Community
- [ ] I have read the [CONTRIBUTING](CONTRIBUTING.md) guide
- [ ] My commit messages follow the project's convention
- [ ] I have linked related issues above
- [ ] I have tested this change thoroughly

## 🎯 Additional Notes
<!-- Add any additional notes, concerns, or questions for reviewers -->

## 🙏 Acknowledgments
<!-- Give credit to anyone who helped with this PR -->
- Thanks to @username for suggestion/help
- Inspired by [project/issue/discussion]

---

**By submitting this PR, I confirm that my contribution is made under the terms of the project's MIT license.**