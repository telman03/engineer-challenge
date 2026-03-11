# AI Usage in This Project

I used **GitHub Copilot (Claude)** in VS Code throughout development — mostly as a pair programming buddy to bounce ideas off of and speed up the boring parts.

## How I Used It

Honestly, the AI was most useful for the stuff I already knew how to do but didn't want to type out manually. Things like wiring up PostgreSQL repositories, writing repetitive test cases, and generating Terraform boilerplate. For the actual architecture — deciding where to split bounded contexts, choosing bcrypt over argon2id, figuring out the CQRS boundaries — that was all me thinking through trade-offs.

### The Workflow

I'd typically start by sketching out what I wanted (mentally or in comments), then ask the AI to help implement it. If something looked wrong or over-engineered, I'd adjust. A few times I threw away what it gave me and started over with a simpler approach.

**Rough timeline:**

1. **Domain modeling** — I worked through the Identity vs Auth context split, defined aggregates and value objects. AI helped flesh out the implementations once I had the structure clear.

2. **Infrastructure + app layer** — This is where AI saved the most time. Repository implementations, CQRS handlers, JWT/bcrypt wrappers — all fairly standard patterns that just need to be written correctly.

3. **IaC & deployment** — Dockerfile, Docker Compose, Terraform, K8s manifests. AI generated the initial versions, I reviewed and tweaked configs.

4. **Tests & frontend** — AI helped generate test cases for value object validation and aggregate invariants. Frontend was straightforward React forms.

5. **Documentation** — ADRs, API docs, architecture diagrams. AI drafted, I edited for accuracy.

## Where AI Helped vs Where It Didn't

**Helped a lot:**
- Generating boilerplate code that follows established patterns
- Keeping code style consistent across dozens of files
- Catching silly mistakes (unused imports, missing error returns)
- Drafting documentation from existing code

**Didn't help much with:**
- Deciding *how* to structure the domain (bounded context boundaries, aggregate design)
- Security trade-offs (hashing algorithm, token strategy, rate limiting approach)
- Knowing when to stop — AI tends to over-engineer if you let it
- Understanding what's "good enough" for this scope vs what's production overkill
