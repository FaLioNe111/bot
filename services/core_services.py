from datetime import datetime, timedelta
from typing import List, Optional, Dict
import re

from utils.date_parser import parse_natural_date, parse_natural_time, parse_duration, extract_event_info
from services.storage import storage


class CalendarService:
    """Service for managing calendar events."""
    
    def create_event_from_text(self, user_id: str, text: str) -> tuple[dict, Optional[str]]:
        """Create event from natural language. Returns (event, error_message)."""
        # Check limits
        limit_check = storage.check_free_limits(user_id)
        if not limit_check["ok"]:
            return None, self._get_limit_error(limit_check)
        
        # Parse natural language
        info = extract_event_info(text)
        
        if not info["date"]:
            return None, "Не удалось определить дату. Укажите дату явно (например, 'завтра' или '15 июня')."
        
        # Build event data
        event_data = {
            "title": info["title"],
            "date": info["date"].isoformat(),
            "time": info["time"],
            "duration_min": info["duration"] or 60,
            "location": info["location"],
            "category": info["category"] or "personal",
            "recurrence": {"type": "none", "interval": 1, "days_of_week": []},
            "reminder_offsets": [],
        }
        
        event = storage.create_event(user_id, event_data)
        return event, None
    
    def check_conflicts(self, user_id: str, date: str, time: str, duration_min: int = 60) -> List[dict]:
        """Check for scheduling conflicts."""
        events = storage.get_user_events(user_id)
        conflicts = []
        
        event_start = datetime.fromisoformat(f"{date}T{time}")
        event_end = event_start + timedelta(minutes=duration_min)
        
        for event in events:
            if event.get("date") != date:
                continue
            if not event.get("time"):
                continue
            
            existing_start = datetime.fromisoformat(f"{date}T{event['time']}")
            existing_end = existing_start + timedelta(minutes=event.get("duration_min", 60))
            
            # Check overlap
            if event_start < existing_end and event_end > existing_start:
                conflicts.append(event)
        
        return conflicts
    
    def get_day_schedule(self, user_id: str, date: datetime) -> List[dict]:
        """Get all events for a specific day."""
        return storage.get_user_events(
            user_id,
            start_date=date,
            end_date=date + timedelta(days=1)
        )
    
    def get_week_schedule(self, user_id: str, start_date: datetime = None) -> Dict[str, List[dict]]:
        """Get events for the week starting from start_date."""
        if start_date is None:
            start_date = datetime.now()
        
        # Find Monday of current week
        monday = start_date - timedelta(days=start_date.weekday())
        sunday = monday + timedelta(days=6)
        
        events = storage.get_user_events(user_id, start_date=monday, end_date=sunday)
        
        # Group by date
        schedule = {}
        for event in events:
            date_str = event["date"]
            if date_str not in schedule:
                schedule[date_str] = []
            schedule[date_str].append(event)
        
        return schedule
    
    def find_free_slots(self, user_id: str, date: datetime, duration_min: int = 60) -> List[tuple[str, str]]:
        """Find free time slots on a given day. Returns list of (start_time, end_time)."""
        events = self.get_day_schedule(user_id, date)
        
        # Working hours
        work_start = datetime.strptime("09:00", "%H:%M")
        work_end = datetime.strptime("18:00", "%H:%M")
        
        # Sort events by time
        busy_periods = []
        for event in events:
            if not event.get("time"):
                continue
            start = datetime.strptime(event["time"], "%H:%M")
            end = start + timedelta(minutes=event.get("duration_min", 60))
            busy_periods.append((start, end))
        
        busy_periods.sort()
        
        # Find gaps
        free_slots = []
        current_time = work_start
        
        for busy_start, busy_end in busy_periods:
            if current_time < busy_start:
                # Found a free slot
                slot_end = busy_start
                if (slot_end - current_time).total_seconds() >= duration_min * 60:
                    free_slots.append((
                        current_time.strftime("%H:%M"),
                        slot_end.strftime("%H:%M")
                    ))
            current_time = max(current_time, busy_end)
        
        # Check end of day
        if current_time < work_end:
            remaining = work_end - current_time
            if remaining.total_seconds() >= duration_min * 60:
                free_slots.append((
                    current_time.strftime("%H:%M"),
                    work_end.strftime("%H:%M")
                ))
        
        return free_slots
    
    def _get_limit_error(self, limit_check: dict) -> str:
        limit = limit_check["limit"]
        current = limit_check["current"]
        max_val = limit_check["max"]
        
        messages = {
            "events": f"📊 Ты использовал {current}/{max_val} событий на Free-плане.",
            "notes": f"📊 Ты использовал {current}/{max_val} заметок на Free-плане.",
            "reminders": f"📊 Ты использовал {current}/{max_val} напоминаний на Free-плане.",
            "habits": f"📊 Ты используешь {current}/{max_val} привычек на Free-плане.",
        }
        
        return messages.get(limit, "Достигнут лимит Free-плана.")


class NoteService:
    """Service for managing notes."""
    
    def create_note(self, user_id: str, title: str, body: str, note_type: str = "text", 
                    tags: List[str] = None, folder: str = None) -> tuple[dict, Optional[str]]:
        """Create a new note."""
        # Check limits
        limit_check = storage.check_free_limits(user_id)
        if not limit_check["ok"]:
            return None, self._get_limit_error(limit_check)
        
        # Auto-detect checklist type
        if note_type == "text" and body.strip().startswith(("-", "•", "1.")):
            lines = [l.strip() for l in body.split("\n") if l.strip()]
            if all(l.startswith(("-", "•")) or l[0].isdigit() for l in lines):
                note_type = "checklist"
                body = [{"text": re.sub(r"^[-•]\s*|\d+\.\s*", "", l), "done": False} for l in lines]
        
        note_data = {
            "title": title,
            "body": body,
            "type": note_type,
            "tags": tags or [],
            "folder": folder,
            "pinned": False,
            "archived": False,
        }
        
        note = storage.create_note(user_id, note_data)
        return note, None
    
    def search_notes(self, user_id: str, query: str) -> List[dict]:
        """Search notes by keyword."""
        notes = storage.get_user_notes(user_id)
        query_lower = query.lower()
        
        results = []
        for note in notes:
            if query_lower in note["title"].lower():
                results.append(note)
            elif isinstance(note["body"], str) and query_lower in note["body"].lower():
                results.append(note)
            elif any(query_lower in tag.lower() for tag in note.get("tags", [])):
                results.append(note)
        
        return results
    
    def _get_limit_error(self, limit_check: dict) -> str:
        return f"📊 Ты использовал {limit_check['current']}/{limit_check['max']} заметок на Free-плане."


class ReminderService:
    """Service for managing reminders."""
    
    def create_reminder(self, user_id: str, message: str, trigger_time: datetime,
                       target_type: str = "standalone", target_id: str = None) -> tuple[dict, Optional[str]]:
        """Create a reminder."""
        # Check limits
        limit_check = storage.check_free_limits(user_id)
        if not limit_check["ok"]:
            return None, self._get_limit_error(limit_check)
        
        reminder_data = {
            "target_type": target_type,
            "target_id": target_id,
            "message": message,
            "trigger_time": trigger_time.isoformat(),
            "status": "pending",
        }
        
        reminder = storage.create_reminder(user_id, reminder_data)
        return reminder, None
    
    def create_reminder_from_text(self, user_id: str, text: str) -> tuple[dict, Optional[str]]:
        """Create reminder from natural language."""
        # Try to parse time
        dt = parse_natural_date(text)
        
        if not dt:
            return None, "Не удалось определить время напоминания."
        
        # Extract message (everything after time expression)
        message = text
        for word in ["напомни", "через", "в", "завтра", "сегодня"]:
            if word in message.lower():
                message = message.split(word, 1)[-1].strip()
                break
        
        return self.create_reminder(user_id, message, dt)
    
    def snooze_reminder(self, reminder_id: str, minutes: int) -> dict:
        """Snooze a reminder."""
        reminder = storage.get_reminder(reminder_id)
        if reminder:
            new_time = datetime.fromisoformat(reminder["trigger_time"]) + timedelta(minutes=minutes)
            storage.update_reminder(reminder_id, trigger_time=new_time.isoformat(), status="snoozed")
        return reminder
    
    def _get_limit_error(self, limit_check: dict) -> str:
        return f"📊 Ты использовал {limit_check['current']}/{limit_check['max']} напоминаний на Free-плане."


class HabitService:
    """Service for managing habits."""
    
    def create_habit(self, user_id: str, name: str, icon: str = "🎯", 
                     frequency: str = "daily", reminder_time: str = None) -> tuple[dict, Optional[str]]:
        """Create a new habit."""
        # Check limits
        limit_check = storage.check_free_limits(user_id)
        if not limit_check["ok"]:
            return None, self._get_limit_error(limit_check)
        
        habit_data = {
            "name": name,
            "icon": icon,
            "frequency": frequency,
            "reminder_time": reminder_time,
            "target_days": list(range(7)) if frequency == "daily" else [],
        }
        
        habit = storage.create_habit(user_id, habit_data)
        return habit, None
    
    def complete_habit(self, user_id: str, habit_id: str, date: datetime = None) -> dict:
        """Mark habit as completed for today."""
        if date is None:
            date = datetime.now()
        return storage.complete_habit(habit_id, date)
    
    def get_habit_stats(self, user_id: str) -> dict:
        """Get overall habit statistics."""
        habits = storage.get_user_habits(user_id)
        
        if not habits:
            return {"total": 0, "completed_today": 0, "best_streak": 0}
        
        today = datetime.now().date().isoformat()
        completed_today = sum(1 for h in habits if today in h.get("completions", []))
        best_streak = max((h.get("best_streak", 0) for h in habits), default=0)
        
        return {
            "total": len(habits),
            "completed_today": completed_today,
            "best_streak": best_streak,
        }
    
    def _get_limit_error(self, limit_check: dict) -> str:
        return f"📊 Ты используешь {limit_check['current']}/{limit_check['max']} привычек на Free-плане."


# Global service instances
calendar_service = CalendarService()
note_service = NoteService()
reminder_service = ReminderService()
habit_service = HabitService()
