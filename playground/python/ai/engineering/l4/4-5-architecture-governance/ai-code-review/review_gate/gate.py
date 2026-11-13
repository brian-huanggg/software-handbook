"""The merge gate: automated results first, then human approvals."""

from dataclasses import dataclass

from review_gate.policy import Change, classify


@dataclass(frozen=True)
class Approval:
    reviewer: str
    commit: str
    is_bot: bool = False


@dataclass(frozen=True)
class PullRequest:
    change: Change
    head_commit: str
    approvals: tuple[Approval, ...]
    checks_green: bool
    open_findings: int


@dataclass(frozen=True)
class Decision:
    can_merge: bool
    reason: str


def valid_approvers(pr: PullRequest) -> set[str]:
    """People whose approval counts: a human, not the author, on the current head commit."""
    return {
        a.reviewer
        for a in pr.approvals
        if a.commit == pr.head_commit and not a.is_bot and a.reviewer != pr.change.author
    }


def decide(pr: PullRequest, trusted_authors: frozenset[str]) -> Decision:
    if not pr.checks_green:
        return Decision(False, "automated checks are not green")
    if pr.open_findings > 0:
        return Decision(False, f"{pr.open_findings} review findings are still open")

    tier = classify(pr.change, trusted_authors)
    have = len(valid_approvers(pr))
    if have < tier:
        need, short = int(tier), pr.head_commit[:7]
        return Decision(False, f"{tier.name} tier needs {need} approvals on {short}, has {have}")
    return Decision(True, f"{tier.name} tier, {have}/{int(tier)} human approvals")
