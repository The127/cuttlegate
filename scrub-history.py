#!/usr/bin/env python3
"""
Scrub team/claude/meta/case-study references from entire git history.

Run: python3 scrub-history.py
"""
import re
import sys
import os

sys.path.insert(0, os.path.join(os.path.expanduser("~"), ".local/lib/python3.13/site-packages"))
import git_filter_repo as fr

# ---------------------------------------------------------------------------
# Blob content replacements (order matters — specific before generic)
# ---------------------------------------------------------------------------
BLOB_REPLACEMENTS = [
    # --- Go code comments ---
    (rb"// This type is Vera's\. Do not touch it unless she says so\.\n", b""),
    (rb"//   - BDD scenario execution \(that is Omar's domain\)\n", b""),

    # --- CSS comment ---
    (rb"Light-mode token overrides \(Mia's spec\)", b"Light-mode token overrides"),

    # --- ui-design.md ---
    (rb"\*\*Owner: Mia\*\*\n", b""),
    (rb"Mia is responsible for keeping it current\. ?", b""),
    (rb"Mia blocks it at review\.", b"the reviewer blocks it."),
    (rb"Design sign-off is required", b"Design sign-off is required"),

    # --- security-checklist ---
    (rb" Auditor: Kai\. Reviewed by: Klaus, Vera, Raj\.", b""),
    (rb"\*\*Kai's determination:\*\*", b"**Determination:**"),
    (rb"\*\*Kai sign-off:\*\*", b"**Security sign-off:**"),

    # --- api-contract.md ---
    (rb"Agreed — Vera \(Staff Architect\) \+ Mia \(Frontend Engineer\)",
     b"Agreed"),
    (rb"sign-off from architecture and frontend",
     b"sign-off from architecture and frontend"),
    (rb"\| Vera \+ Mia \|", b"|  |"),

    # --- ADR 0004 (heavy — Claude/team context system references) ---
    (rb"structured team persona files in `\.claude/team/`, `CLAUDE\.md`, and ADRs in `docs/adr/`",
     b"ADRs in `docs/adr/`"),
    (rb"file-based memory in `\.claude/projects/\.\.\./memory/` with an indexed `MEMORY\.md`, ",
     b""),
    (rb"The existing context system \(file-based memory, CLAUDE\.md, ADRs, team files\) is working",
     b"The existing context system is working"),
    (rb"The existing file-based memory and CLAUDE\.md system remains the agent context mechanism",
     b"The existing context system remains the agent context mechanism"),

    # --- ADR 0012 ---
    (rb"the team identified", b"the team identified"),

    # --- ADR 0017 ---
    (rb"Security review raised that", b"Security review raised that"),

    # --- ADR 0029 ---
    (rb"and explicit security sign-off", b"and explicit security sign-off"),
    (rb"explicit security sign-off", b"explicit security sign-off"),

    # --- VISION.md ---
    (rb"## Project owner's role", b"## Project owner's role"),
    (rb"The project owner owns them and updates them",
     b"The project owner owns them and updates them"),
    (rb"Klaus changes it\.", b"the project owner changes it."),
    (rb"Klaus holds this vision between sessions\. He reads this file at the start of every session\. He has authority to:",
     b"The project owner holds this vision between sessions and has authority to:"),
    (rb"The project owner does not make", b"The project owner does not make"),
    (rb"They surface them", b"They surface them"),

    # --- engineering-principles.md ---
    (rb"For AI-specific process rules, see `CLAUDE\.md`\. ", b""),
    (rb"_This document complements but does not duplicate `CLAUDE\.md` \(AI process rules\) or `docs/adr/` \(individual decisions\)\.",
     b"_This document complements but does not duplicate `docs/adr/` (individual decisions)."),

    # --- README.md ---
    (rb"See \[CLAUDE\.md\]\(CLAUDE\.md\) for architecture guide and contribution conventions\.\n", b""),

    # --- onboarding-validation-report.md ---
    (rb"## Documentation sign-off", b"## Documentation sign-off"),

    # --- .gitignore ---
    (rb"# Claude Code worktrees\n\.claude/worktrees/\n", b""),
    (rb"\n# Process intelligence[^\n]*\nmeta/\n!meta/harvest-tokens\.py\n", b"\n"),

    # --- Generic team -> team (LAST — after all specific patterns) ---
    (rb"\bcrew\b", b"team"),
    (rb"\bCrew\b", b"Team"),
]

BLOB_COMPILED = [(re.compile(p), r) for p, r in BLOB_REPLACEMENTS]

# ---------------------------------------------------------------------------
# Commit message replacements
# ---------------------------------------------------------------------------
MSG_REPLACEMENTS = [
    # Specific multi-word patterns first
    (rb"Lena cold[- ]?read", b"cold read"),
    (rb" and add Finn to Go SDK grooming", b""),
    (rb" — ADR 0027, Priya persona, CLAUDE\.md", b" — ADR 0027"),
    (rb", Priya persona", b""),
    (rb"Priya persona, ", b""),
    (rb" add proactive engagement rules to Finn and Mia", b" add proactive engagement rules"),
    (rb" elevate Mia to design authority and add ui-design\.md", b" add ui-design.md"),
    (rb" sharpen Lena and Raj on idiom vs readability tension",
     b" sharpen idiom vs readability tension"),
    (rb" add Finn persona, update Mia and Omar, document git workflow",
     b" update personas, document git workflow"),
    (rb" sharpen Finn persona — proactive on domain-to-SDK naming, challenge not confirm",
     b" sharpen SDK persona — proactive on domain-to-SDK naming"),
    (rb" sharpen Klaus — sprint goal ownership, user value framing, estimate challenge",
     b" sharpen sprint goal ownership, user value framing, estimate challenge"),
    (rb" add Voice section to Klaus — one challenge, one sentence, then stop",
     b" refine persona voice"),
    (rb" soften Klaus voice — complete natural sentences, no rambling",
     b" refine persona voice — complete natural sentences"),
    (rb" add Lena cold-read gate for doc issues before in_review",
     b" add cold-read gate for doc issues before in_review"),

    # Scopes
    (rb"docs\(team\)", b"docs"),
    (rb"chore\(team\)", b"chore"),
    (rb"docs\(claude\)", b"docs"),
    (rb"chore\(claude\)", b"chore"),

    # CLAUDE.md references
    (rb"CLAUDE\.md", b"project docs"),
    (rb"CLAUDE.md", b"project docs"),

    # Individual names (after specific patterns)
    (rb"\bLena\b", b""),
    (rb"\bFinn\b", b""),
    (rb"\bMia\b", b""),
    (rb"\bKlaus\b", b""),
    (rb"\bOmar\b", b""),
    (rb"\bRaj\b", b""),
    (rb"\bVera\b", b""),
    (rb"\bKai\b", b""),
    (rb"\bAmara\b", b""),
    (rb"\bPriya\b", b""),
    (rb"\bSeo\b", b""),

    # Generic
    (rb"\bcrew\b", b"team"),
    (rb"\bCrew\b", b"Team"),

    # Cleanup artifacts
    (rb"  +", b" "),
    (rb" +\n", b"\n"),
    (rb": +", b": "),
    (rb" — \n", b"\n"),
    (rb" —\n", b"\n"),
    (rb"add team personas, ADR 0010", b"add ADR 0010"),
    (rb"update team process and decision guidelines",
     b"update process and decision guidelines"),
]

MSG_COMPILED = [(re.compile(p), r) for p, r in MSG_REPLACEMENTS]


def blob_cb(blob, callback_metadata):
    if b'\x00' in blob.data[:8192]:
        return
    data = blob.data
    for pattern, replacement in BLOB_COMPILED:
        data = pattern.sub(replacement, data)
    blob.data = data


def commit_cb(commit, callback_metadata):
    msg = commit.message
    for pattern, replacement in MSG_COMPILED:
        msg = pattern.sub(replacement, msg)
    commit.message = msg


def main():
    args = fr.FilteringOptions.default_options()
    args.force = True
    args.partial = True

    filt = fr.RepoFilter(
        args,
        blob_callback=blob_cb,
        commit_callback=commit_cb,
    )
    filt.run()


if __name__ == "__main__":
    main()
