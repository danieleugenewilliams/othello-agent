# Documentation Guide
## How to Use Othello Documentation

**Last Updated:** October 10, 2025

---

## Quick Reference

### For Understanding the Project

**Start Here:**
1. **[README.md](../README.md)** - Project overview and quick start
2. **[PRD.md](./PRD.md)** - Product vision and goals
3. **[ARCHITECTURE.md](./ARCHITECTURE.md)** - System design and components

### For Development

**Current Implementation:**
- **[TDD_IMPLEMENTATION_PLAN.md](./TDD_IMPLEMENTATION_PLAN.md)** 🧪 - **USE THIS DAILY**
  - Day-by-day tasks with tests
  - Copy/paste ready test code
  - Implementation examples
  - Acceptance checklist

**Strategic Planning:**
- **[MCP_TUI_INTEGRATION.md](./MCP_TUI_INTEGRATION.md)** ⚡ - **USE FOR CONTEXT**
  - Week-by-week overview
  - Architecture decisions
  - Risk mitigation
  - Success criteria

**Reference:**
- **[IMPLEMENTATION.md](./IMPLEMENTATION.md)** - General implementation patterns
- **[REQUIREMENTS.md](./REQUIREMENTS.md)** - Functional requirements
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - Component design

### For Users

**Getting Started:**
- **[USAGE.md](./USAGE.md)** - Installation and configuration
- **[README.md](../README.md)** - Quick start guide

---

## Documentation Workflow

### Daily Development Flow

```
1. Check TDD_IMPLEMENTATION_PLAN.md for today's tasks
   ↓
2. Copy test code from plan
   ↓
3. Write test (Red)
   ↓
4. Copy/adapt implementation code
   ↓
5. Make test pass (Green)
   ↓
6. Refactor
   ↓
7. Check acceptance criteria
   ↓
8. Commit and move to next task
```

### Weekly Planning Flow

```
1. Review MCP_TUI_INTEGRATION.md for week's goals
   ↓
2. Check week's success criteria
   ↓
3. Follow daily tasks in TDD_IMPLEMENTATION_PLAN.md
   ↓
4. Run integration tests at week end
   ↓
5. Update progress in README.md
```

---

## Document Purposes

### PRD.md
**When to use:** Understanding why we're building Othello
- Product vision and market analysis
- User stories and personas
- Success metrics
- Competitive landscape

**Don't use for:** Technical implementation details

### REQUIREMENTS.md
**When to use:** Validating features and acceptance criteria
- Functional requirements checklist
- Non-functional requirements
- System requirements
- Testing requirements

**Don't use for:** How to implement features

### ARCHITECTURE.md
**When to use:** Understanding system design
- Component interactions
- Data flow diagrams
- Design patterns
- Security model

**Don't use for:** Step-by-step implementation

### IMPLEMENTATION.md
**When to use:** Reference for coding patterns
- Go best practices
- Project structure
- Code examples
- Testing patterns

**Don't use for:** Current work-in-progress tasks

### MCP_TUI_INTEGRATION.md ⚡
**When to use:** Understanding the big picture
- Weekly goals and strategy
- Integration architecture
- Risk assessment
- Timeline overview

**Don't use for:** Daily task execution

### TDD_IMPLEMENTATION_PLAN.md 🧪
**When to use:** Daily development work
- Exact tests to write
- Implementation code to write
- Today's tasks
- Acceptance criteria

**This is your primary working document!**

### USAGE.md
**When to use:** User-facing documentation
- Installation instructions
- Configuration guide
- Usage examples
- Troubleshooting

**Don't use for:** Development guidance

---

## Document Relationships

```
PRD.md (Why?)
    ↓
REQUIREMENTS.md (What?)
    ↓
ARCHITECTURE.md (How at high level?)
    ↓
MCP_TUI_INTEGRATION.md (What's the strategy?)
    ↓
TDD_IMPLEMENTATION_PLAN.md (What do I do today?)
    ↓
IMPLEMENTATION.md (What patterns should I use?)
    ↓
Code Implementation
    ↓
USAGE.md (How do users use it?)
```

---

## Current Focus: MCP-TUI Integration

### Primary Documents (October 2025)

**Daily Work:**
- 📖 **TDD_IMPLEMENTATION_PLAN.md** - Your daily checklist

**Context:**
- 📖 **MCP_TUI_INTEGRATION.md** - Understanding the integration

**Reference:**
- 📖 **ARCHITECTURE.md** - Component design
- 📖 **IMPLEMENTATION.md** - Coding patterns

### Current Week: Week 1 - Agent-MCP Integration

**Today's Focus:**
1. Open `TDD_IMPLEMENTATION_PLAN.md`
2. Find current day's tasks
3. Copy test code
4. Write tests (Red)
5. Implement (Green)
6. Refactor
7. Check off acceptance criteria

---

## Quick Links by Role

### 👨‍💻 Developer (Implementation)
1. [TDD_IMPLEMENTATION_PLAN.md](./TDD_IMPLEMENTATION_PLAN.md) ← Start here
2. [MCP_TUI_INTEGRATION.md](./MCP_TUI_INTEGRATION.md)
3. [ARCHITECTURE.md](./ARCHITECTURE.md)
4. [IMPLEMENTATION.md](./IMPLEMENTATION.md)

### 🏗️ Architect (Design)
1. [ARCHITECTURE.md](./ARCHITECTURE.md)
2. [MCP_TUI_INTEGRATION.md](./MCP_TUI_INTEGRATION.md)
3. [REQUIREMENTS.md](./REQUIREMENTS.md)

### 📋 Product Manager (Features)
1. [PRD.md](./PRD.md)
2. [REQUIREMENTS.md](./REQUIREMENTS.md)
3. [MCP_TUI_INTEGRATION.md](./MCP_TUI_INTEGRATION.md)

### 👤 User (Usage)
1. [USAGE.md](./USAGE.md)
2. [README.md](../README.md)
3. [Troubleshooting](./USAGE.md#troubleshooting)

### ✅ QA (Testing)
1. [TDD_IMPLEMENTATION_PLAN.md](./TDD_IMPLEMENTATION_PLAN.md) - Test cases
2. [REQUIREMENTS.md](./REQUIREMENTS.md) - Acceptance criteria
3. [Integration Tests](../integration_test.go)

---

## Update Frequency

| Document | Update Frequency | When to Update |
|----------|------------------|----------------|
| TDD_IMPLEMENTATION_PLAN.md | Daily | After completing tasks |
| MCP_TUI_INTEGRATION.md | Weekly | After completing week |
| README.md | Weekly | Major milestones |
| PRD.md | Monthly | Product changes |
| REQUIREMENTS.md | Bi-weekly | Requirement changes |
| ARCHITECTURE.md | As needed | Design changes |
| IMPLEMENTATION.md | As needed | Pattern additions |
| USAGE.md | Weekly | Feature additions |

---

## Document Status Icons

- ✅ **Complete** - Stable, no active changes
- ⚡ **Active** - Currently being used for implementation
- 🧪 **Active** - Daily development work
- 📝 **Draft** - Work in progress
- 🔄 **Updating** - Being revised
- ⏸️ **Paused** - On hold

---

## Tips for Effective Documentation Use

### For Developers

1. **Start your day with TDD_IMPLEMENTATION_PLAN.md**
   - Find your current task
   - Copy the test code
   - Follow Red-Green-Refactor

2. **Reference MCP_TUI_INTEGRATION.md for context**
   - Understand why you're building this
   - See how it fits into the bigger picture

3. **Check ARCHITECTURE.md when confused**
   - Understand component relationships
   - Verify design patterns

4. **Update progress as you go**
   - Check off completed items
   - Note any blockers
   - Update status in README

### For Code Review

1. Check against REQUIREMENTS.md for acceptance criteria
2. Verify design follows ARCHITECTURE.md patterns
3. Ensure tests exist per TDD_IMPLEMENTATION_PLAN.md
4. Validate code style from IMPLEMENTATION.md

### For Planning

1. Review PRD.md for product direction
2. Check REQUIREMENTS.md for must-haves
3. Update MCP_TUI_INTEGRATION.md strategy
4. Break down into TDD_IMPLEMENTATION_PLAN.md tasks

---

## Getting Help

### Where to Look

**Question Type** → **Document**
- "What should I work on today?" → TDD_IMPLEMENTATION_PLAN.md
- "Why are we building this?" → PRD.md
- "What are the requirements?" → REQUIREMENTS.md
- "How does X component work?" → ARCHITECTURE.md
- "How do I implement Y?" → IMPLEMENTATION.md
- "How do users do Z?" → USAGE.md

### Still Stuck?

1. Check the relevant document's Table of Contents
2. Search for keywords in the document
3. Look at code examples in IMPLEMENTATION.md
4. Review test examples in TDD_IMPLEMENTATION_PLAN.md
5. Open an issue or ask the team

---

## Contributing to Documentation

### When to Update Documentation

- **Code changes** → Update TDD_IMPLEMENTATION_PLAN.md progress
- **Feature complete** → Update USAGE.md with examples
- **Design change** → Update ARCHITECTURE.md
- **New pattern** → Update IMPLEMENTATION.md
- **Requirement change** → Update REQUIREMENTS.md
- **Product pivot** → Update PRD.md

### How to Update

1. Make changes in your branch
2. Keep changes focused and specific
3. Update "Last Updated" date
4. Update Document Status table in README.md
5. Submit PR with documentation changes

---

*This guide helps you navigate Othello's documentation efficiently. When in doubt, start with TDD_IMPLEMENTATION_PLAN.md for daily work!*
