from datetime import datetime, date, time
from typing import Optional, Literal
from pydantic import BaseModel, Field


class Recurrence(BaseModel):
    type: Literal["none", "daily", "weekly", "monthly", "yearly", "custom"] = "none"
    interval: int = 1
    days_of_week: list[int] = []  # 0=Monday, 6=Sunday
    end_date: Optional[date] = None


class Event(BaseModel):
    id: str
    title: str
    date: date
    time: Optional[time] = None
    duration_min: int = 60
    end_date: Optional[date] = None
    location: Optional[str] = None
    description: Optional[str] = None
    category: str = "personal"
    recurrence: Recurrence = Field(default_factory=Recurrence)
    reminder_offsets: list[int] = []  # minutes before event
    linked_note_id: Optional[str] = None
    created_at: datetime
    updated_at: datetime


class NoteType(str):
    TEXT = "text"
    CHECKLIST = "checklist"
    STRUCTURED = "structured"


class ChecklistItem(BaseModel):
    text: str
    done: bool = False


class Note(BaseModel):
    id: str
    title: str
    body: str | list[ChecklistItem]
    type: NoteType = NoteType.TEXT
    tags: list[str] = []
    folder: Optional[str] = None
    pinned: bool = False
    archived: bool = False
    linked_event_id: Optional[str] = None
    created_at: datetime
    updated_at: datetime


class ReminderStatus(str):
    PENDING = "pending"
    SENT = "sent"
    DONE = "done"
    SNOOZED = "snoozed"


class Reminder(BaseModel):
    id: str
    target_type: Literal["event", "note", "standalone"]
    target_id: Optional[str] = None
    message: Optional[str] = None
    trigger_time: datetime
    recurrence: Recurrence = Field(default_factory=Recurrence)
    snoozed_until: Optional[datetime] = None
    status: ReminderStatus = ReminderStatus.PENDING


class Habit(BaseModel):
    id: str
    name: str
    icon: str = "🎯"
    frequency: Literal["daily", "weekly", "custom"] = "daily"
    target_days: list[int] = []  # 0=Monday, 6=Sunday
    reminder_time: Optional[time] = None
    completions: list[date] = []
    current_streak: int = 0
    best_streak: int = 0
    created_at: datetime


class PlanType(str):
    FREE = "free"
    PRO = "pro"
    TRIAL = "trial"


class User(BaseModel):
    id: str
    telegram_id: int
    name: str
    language: str = "ru"
    timezone: str = "Europe/Moscow"
    plan: PlanType = PlanType.FREE
    trial_ends_at: Optional[datetime] = None
    briefing_time: Optional[time] = None
    ads_enabled: bool = True
    referral_code: str
    referred_by: Optional[str] = None
    created_at: datetime
