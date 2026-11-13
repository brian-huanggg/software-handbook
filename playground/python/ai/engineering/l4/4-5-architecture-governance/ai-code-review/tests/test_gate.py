from review_gate.gate import Approval, PullRequest, decide
from review_gate.policy import Change, FileChange

TRUSTED = frozenset({"alice"})
HEAD = "a1b2c3d4e5"


def pr(
    *paths: str,
    approvals: tuple[Approval, ...] = (),
    checks_green: bool = True,
    open_findings: int = 0,
    head: str = HEAD,
) -> PullRequest:
    change = Change("alice", tuple(FileChange(p, 10, 0) for p in paths))
    return PullRequest(change, head, approvals, checks_green, open_findings)


def test_low_risk_merges_without_a_human() -> None:
    assert decide(pr("docs/a.md"), TRUSTED).can_merge


def test_red_checks_block_even_low_risk() -> None:
    assert not decide(pr("docs/a.md", checks_green=False), TRUSTED).can_merge


def test_open_finding_blocks() -> None:
    assert not decide(pr("docs/a.md", open_findings=1), TRUSTED).can_merge


def test_medium_needs_one_human() -> None:
    assert not decide(pr("prompts/system.md"), TRUSTED).can_merge
    ok = pr("prompts/system.md", approvals=(Approval("bob", HEAD),))
    assert decide(ok, TRUSTED).can_merge


def test_approval_on_old_commit_does_not_count() -> None:
    stale = pr("prompts/system.md", approvals=(Approval("bob", "old0000"),))
    assert not decide(stale, TRUSTED).can_merge


def test_bot_approval_does_not_fill_a_human_slot() -> None:
    bot = pr("prompts/system.md", approvals=(Approval("review-bot", HEAD, is_bot=True),))
    assert not decide(bot, TRUSTED).can_merge


def test_author_cannot_approve_own_change() -> None:
    own = pr("prompts/system.md", approvals=(Approval("alice", HEAD),))
    assert not decide(own, TRUSTED).can_merge


def test_high_needs_two_different_people() -> None:
    twice = pr(".github/workflows/ci.yml", approvals=(Approval("bob", HEAD), Approval("bob", HEAD)))
    assert not decide(twice, TRUSTED).can_merge
    two = pr(".github/workflows/ci.yml", approvals=(Approval("bob", HEAD), Approval("carol", HEAD)))
    assert decide(two, TRUSTED).can_merge
