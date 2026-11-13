"""Risk tiering: how many humans must approve a change."""

from dataclasses import dataclass
from enum import IntEnum
from fnmatch import fnmatch


class Tier(IntEnum):
    """The value is the number of human approvals the tier requires."""

    LOW = 0
    MEDIUM = 1
    HIGH = 2


# Paths of the 手冊問答助理 repo. The highest matching tier wins, so order does not matter.
HIGH_PATHS = (".github/workflows/*", "deploy/*", "app/auth/*")
MEDIUM_PATHS = ("prompts/*", "app/*", "evals/*")
LOW_PATHS = ("docs/*", "*.md", "tests/*")


@dataclass(frozen=True)
class FileChange:
    path: str
    added: int
    deleted: int


@dataclass(frozen=True)
class Change:
    author: str
    files: tuple[FileChange, ...]

    @property
    def lines(self) -> int:
        return sum(f.added + f.deleted for f in self.files)


def tier_of_path(path: str) -> Tier:
    if any(fnmatch(path, p) for p in HIGH_PATHS):
        return Tier.HIGH
    if any(fnmatch(path, p) for p in MEDIUM_PATHS):
        return Tier.MEDIUM
    if any(fnmatch(path, p) for p in LOW_PATHS):
        return Tier.LOW
    # A path nobody classified is a path nobody thought about: ask a human.
    return Tier.MEDIUM


def classify(change: Change, trusted_authors: frozenset[str], big_change_lines: int = 400) -> Tier:
    """The change sets the tier. The author only decides who may use the lighter path."""
    tier = max((tier_of_path(f.path) for f in change.files), default=Tier.MEDIUM)
    if change.lines > big_change_lines:
        tier = max(tier, Tier.MEDIUM)
    if change.author not in trusted_authors:
        tier = max(tier, Tier.MEDIUM)
    return tier
