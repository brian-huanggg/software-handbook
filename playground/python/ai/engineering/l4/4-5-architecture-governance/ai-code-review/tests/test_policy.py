import pytest

from review_gate.policy import Change, FileChange, Tier, classify, tier_of_path

TRUSTED = frozenset({"alice"})


def change(*paths: str, author: str = "alice", lines: int = 10) -> Change:
    return Change(author, tuple(FileChange(p, lines, 0) for p in paths))


@pytest.mark.parametrize(
    ("path", "tier"),
    [
        ("docs/intro.md", Tier.LOW),
        ("README.md", Tier.LOW),
        ("prompts/system.md", Tier.MEDIUM),
        ("app/qa.py", Tier.MEDIUM),
        (".github/workflows/ci.yml", Tier.HIGH),
        ("app/auth/token.py", Tier.HIGH),
        ("scripts/unknown.sh", Tier.MEDIUM),
    ],
)
def test_tier_of_path(path: str, tier: Tier) -> None:
    assert tier_of_path(path) == tier


def test_docs_only_change_by_trusted_author_is_low() -> None:
    assert classify(change("docs/a.md", "docs/b.md"), TRUSTED) == Tier.LOW


def test_highest_path_wins() -> None:
    assert classify(change("docs/a.md", "deploy/k3s.yaml"), TRUSTED) == Tier.HIGH


def test_untrusted_author_never_gets_the_light_path() -> None:
    assert classify(change("docs/a.md", author="agent-7"), TRUSTED) == Tier.MEDIUM


def test_big_change_needs_a_human() -> None:
    assert classify(change("docs/a.md", lines=500), TRUSTED) == Tier.MEDIUM


def test_empty_change_is_not_low() -> None:
    assert classify(change(), TRUSTED) == Tier.MEDIUM
