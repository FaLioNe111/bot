from datetime import datetime, timedelta
from typing import Optional, Dict, List
import json


class InMemoryStorage:
    """Simple in-memory storage for development. Replace with real DB in production."""
    
    def __init__(self):
        self.users: Dict[str, dict] = {}
        self.events: Dict[str, dict] = {}
        self.notes: Dict[str, dict] = {}
        self.reminders: Dict[str, dict] = {}
        self.habits: Dict[str, dict] = {}
        
        # Index by user
        self.user_events: Dict[str, List[str]] = {}
        self.user_notes: Dict[str, List[str]] = {}
        self.user_reminders: Dict[str, List[str]] = {}
        self.user_habits: Dict[str, List[str]] = {}
    
    def create_user(self, user_id: str, telegram_id: int, name: str) -> dict:
        from uuid import uuid4
        user = {
            "id": user_id or str(uuid4()),
            "telegram_id": telegram_id,
            "name": name,
            "language": "ru",
            "timezone": "Europe/Moscow",
            "plan": "free",
            "trial_ends_at": None,
            "briefing_time": None,
            "ads_enabled": True,
            "referral_code": str(uuid4())[:8],
            "referred_by": None,
            "created_at": datetime.now().isoformat(),
        }
        self.users[user["id"]] = user
        self.user_events[user["id"]] = []
        self.user_notes[user["id"]] = []
        self.user_reminders[user["id"]] = []
        self.user_habits[user["id"]] = []
        return user
    
    def get_user(self, user_id: str) -> Optional[dict]:
        return self.users.get(user_id)
    
    def get_user_by_telegram_id(self, telegram_id: int) -> Optional[dict]:
        for user in self.users.values():
            if user["telegram_id"] == telegram_id:
                return user
        return None
    
    def update_user(self, user_id: str, **kwargs) -> dict:
        user = self.users.get(user_id)
        if user:
            user.update(kwargs)
            user["updated_at"] = datetime.now().isoformat()
        return user
    
    # Events
    def create_event(self, user_id: str, event_data: dict) -> dict:
        from uuid import uuid4
        event = {
            "id": str(uuid4()),
            "user_id": user_id,
            **event_data,
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat(),
        }
        self.events[event["id"]] = event
        self.user_events.setdefault(user_id, []).append(event["id"])
        return event
    
    def get_event(self, event_id: str) -> Optional[dict]:
        return self.events.get(event_id)
    
    def get_user_events(self, user_id: str, start_date: datetime = None, end_date: datetime = None) -> List[dict]:
        events = []
        for event_id in self.user_events.get(user_id, []):
            event = self.events.get(event_id)
            if event:
                event_date = event.get("date")
                if isinstance(event_date, str):
                    event_date = datetime.fromisoformat(event_date).date()
                
                if start_date and event_date < start_date.date():
                    continue
                if end_date and event_date > end_date.date():
                    continue
                events.append(event)
        return sorted(events, key=lambda x: (x.get("date"), x.get("time") or ""))
    
    def update_event(self, event_id: str, **kwargs) -> dict:
        event = self.events.get(event_id)
        if event:
            event.update(kwargs)
            event["updated_at"] = datetime.now().isoformat()
        return event
    
    def delete_event(self, event_id: str) -> bool:
        event = self.events.pop(event_id, None)
        if event:
            user_id = event.get("user_id")
            if user_id in self.user_events:
                self.user_events[user_id].remove(event_id)
            return True
        return False
    
    # Notes
    def create_note(self, user_id: str, note_data: dict) -> dict:
        from uuid import uuid4
        note = {
            "id": str(uuid4()),
            "user_id": user_id,
            **note_data,
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat(),
        }
        self.notes[note["id"]] = note
        self.user_notes.setdefault(user_id, []).append(note["id"])
        return note
    
    def get_note(self, note_id: str) -> Optional[dict]:
        return self.notes.get(note_id)
    
    def get_user_notes(self, user_id: str, archived: bool = False) -> List[dict]:
        notes = []
        for note_id in self.user_notes.get(user_id, []):
            note = self.notes.get(note_id)
            if note and note.get("archived", False) == archived:
                notes.append(note)
        return sorted(notes, key=lambda x: (not x.get("pinned"), x.get("updated_at", "")), reverse=True)
    
    def update_note(self, note_id: str, **kwargs) -> dict:
        note = self.notes.get(note_id)
        if note:
            note.update(kwargs)
            note["updated_at"] = datetime.now().isoformat()
        return note
    
    def delete_note(self, note_id: str) -> bool:
        note = self.notes.pop(note_id, None)
        if note:
            user_id = note.get("user_id")
            if user_id in self.user_notes:
                self.user_notes[user_id].remove(note_id)
            return True
        return False
    
    # Reminders
    def create_reminder(self, user_id: str, reminder_data: dict) -> dict:
        from uuid import uuid4
        reminder = {
            "id": str(uuid4()),
            "user_id": user_id,
            **reminder_data,
            "created_at": datetime.now().isoformat(),
        }
        self.reminders[reminder["id"]] = reminder
        self.user_reminders.setdefault(user_id, []).append(reminder["id"])
        return reminder
    
    def get_reminder(self, reminder_id: str) -> Optional[dict]:
        return self.reminders.get(reminder_id)
    
    def get_pending_reminders(self, user_id: str) -> List[dict]:
        reminders = []
        for reminder_id in self.user_reminders.get(user_id, []):
            reminder = self.reminders.get(reminder_id)
            if reminder and reminder.get("status") == "pending":
                reminders.append(reminder)
        return reminders
    
    def update_reminder(self, reminder_id: str, **kwargs) -> dict:
        reminder = self.reminders.get(reminder_id)
        if reminder:
            reminder.update(kwargs)
        return reminder
    
    def delete_reminder(self, reminder_id: str) -> bool:
        reminder = self.reminders.pop(reminder_id, None)
        if reminder:
            user_id = reminder.get("user_id")
            if user_id in self.user_reminders:
                self.user_reminders[user_id].remove(reminder_id)
            return True
        return False
    
    # Habits
    def create_habit(self, user_id: str, habit_data: dict) -> dict:
        from uuid import uuid4
        habit = {
            "id": str(uuid4()),
            "user_id": user_id,
            **habit_data,
            "current_streak": 0,
            "best_streak": 0,
            "completions": [],
            "created_at": datetime.now().isoformat(),
        }
        self.habits[habit["id"]] = habit
        self.user_habits.setdefault(user_id, []).append(habit["id"])
        return habit
    
    def get_habit(self, habit_id: str) -> Optional[dict]:
        return self.habits.get(habit_id)
    
    def get_user_habits(self, user_id: str) -> List[dict]:
        habits = []
        for habit_id in self.user_habits.get(user_id, []):
            habit = self.habits.get(habit_id)
            if habit:
                habits.append(habit)
        return habits
    
    def complete_habit(self, habit_id: str, date: datetime) -> dict:
        habit = self.habits.get(habit_id)
        if habit:
            date_str = date.date().isoformat()
            if date_str not in habit["completions"]:
                habit["completions"].append(date_str)
                # Recalculate streak
                habit["current_streak"] = self._calculate_streak(habit["completions"])
                habit["best_streak"] = max(habit["best_streak"], habit["current_streak"])
        return habit
    
    def _calculate_streak(self, completions: List[str]) -> int:
        if not completions:
            return 0
        
        dates = sorted([datetime.fromisoformat(d).date() for d in completions], reverse=True)
        today = datetime.now().date()
        
        streak = 0
        for i, date in enumerate(dates):
            expected = today - timedelta(days=i)
            if date == expected:
                streak += 1
            elif date == expected - timedelta(days=1):
                # Yesterday's completion counts if today is not done yet
                streak += 1
            else:
                break
        return streak
    
    def undo_habit_completion(self, habit_id: str, date: datetime) -> dict:
        habit = self.habits.get(habit_id)
        if habit:
            date_str = date.date().isoformat()
            if date_str in habit["completions"]:
                habit["completions"].remove(date_str)
                habit["current_streak"] = self._calculate_streak(habit["completions"])
        return habit
    
    # Check limits
    def check_free_limits(self, user_id: str) -> dict:
        from config.settings import settings
        
        user = self.users.get(user_id)
        if not user or user.get("plan") == "pro":
            return {"ok": True}
        
        # Count usage
        events_count = len(self.user_events.get(user_id, []))
        notes_count = len(self.user_notes.get(user_id, []))
        reminders_count = len(self.user_reminders.get(user_id, []))
        habits_count = len(self.user_habits.get(user_id, []))
        
        limits = settings.FREE_LIMITS
        
        if events_count >= limits["events_per_month"]:
            return {"ok": False, "limit": "events", "current": events_count, "max": limits["events_per_month"]}
        if notes_count >= limits["active_notes"]:
            return {"ok": False, "limit": "notes", "current": notes_count, "max": limits["active_notes"]}
        if reminders_count >= limits["active_reminders"]:
            return {"ok": False, "limit": "reminders", "current": reminders_count, "max": limits["active_reminders"]}
        if habits_count >= limits["habits"]:
            return {"ok": False, "limit": "habits", "current": habits_count, "max": limits["habits"]}
        
        return {"ok": True}


# Global instance for development
storage = InMemoryStorage()
