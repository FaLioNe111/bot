import re
from datetime import datetime, timedelta
from typing import Optional, Tuple
from dateutil import parser as date_parser
from dateutil.relativedelta import relativedelta


def parse_natural_date(text: str, reference_date: datetime = None) -> Optional[datetime]:
    """Parse natural language date expressions."""
    if reference_date is None:
        reference_date = datetime.now()
    
    text = text.lower().strip()
    
    # Today variations
    if text in ["сегодня", "today", "сейчас", "now"]:
        return reference_date
    
    # Tomorrow variations
    if text in ["завтра", "tomorrow"]:
        return reference_date + timedelta(days=1)
    
    # Yesterday
    if text in ["вчера", "yesterday"]:
        return reference_date - timedelta(days=1)
    
    # Day of week (next occurrence)
    weekdays_ru = {
        "понедельник": 0, "пн": 0,
        "вторник": 1, "вт": 1,
        "среда": 2, "ср": 2,
        "четверг": 3, "чт": 3,
        "пятница": 4, "пт": 4,
        "суббота": 5, "сб": 5,
        "воскресенье": 6, "вс": 6,
    }
    
    for day_name, weekday in weekdays_ru.items():
        if day_name in text:
            days_ahead = weekday - reference_date.weekday()
            if days_ahead < 0:  # Target day already happened this week
                days_ahead += 7
            return reference_date + timedelta(days=days_ahead)
    
    # "через N дней/часов/минут"
    match = re.search(r'через\s+(\d+)\s+(день|дня|дней|час|часа|часов|минуту|минуты|минут)', text)
    if match:
        value = int(match.group(1))
        unit = match.group(2)
        if "минут" in unit:
            return reference_date + timedelta(minutes=value)
        elif "час" in unit:
            return reference_date + timedelta(hours=value)
        else:
            return reference_date + timedelta(days=value)
    
    # Try standard date parsing
    try:
        parsed = date_parser.parse(text, fuzzy=True)
        if parsed.year == 1900:  # No year in input
            parsed = parsed.replace(year=reference_date.year)
        return parsed
    except:
        pass
    
    return None


def parse_natural_time(text: str) -> Optional[str]:
    """Extract time from natural language text. Returns HH:MM format."""
    text = text.lower().strip()
    
    # Direct time patterns: 15:00, 15.00, 15-00
    match = re.search(r'(\d{1,2})[:\.\-](\d{2})', text)
    if match:
        hour, minute = int(match.group(1)), int(match.group(2))
        if 0 <= hour <= 23 and 0 <= minute <= 59:
            return f"{hour:02d}:{minute:02d}"
    
    # Hour only: "в 15", "на 10"
    match = re.search(r'[ввна]\s+(\d{1,2})\s*(часов|часа|утра|вечера|дня)?', text)
    if match:
        hour = int(match.group(1))
        modifier = match.group(2) or ""
        
        # Adjust for morning/evening if needed
        if "утра" in modifier and hour == 12:
            hour = 0
        elif "вечера" in modifier and hour < 12:
            hour += 12
        
        if 0 <= hour <= 23:
            return f"{hour:02d}:00"
    
    # Relative times
    if "утром" in text:
        return "09:00"
    if "днём" in text or "днем" in text:
        return "14:00"
    if "вечером" in text:
        return "19:00"
    if "ночью" in text:
        return "22:00"
    
    return None


def parse_duration(text: str) -> Optional[int]:
    """Parse duration in minutes from text."""
    text = text.lower()
    
    match = re.search(r'(\d+)\s*(час|часа|часов|ч|h)', text)
    hours = int(match.group(1)) if match else 0
    
    match = re.search(r'(\d+)\s*(мин|минуты|минут|m)', text)
    minutes = int(match.group(1)) if match else 0
    
    total = hours * 60 + minutes
    return total if total > 0 else None


def parse_recurrence(text: str) -> Tuple[str, int, list[int]]:
    """Parse recurrence pattern from text. Returns (type, interval, days_of_week)."""
    text = text.lower()
    
    days_ru = {
        "понедельник": 0, "пн": 0,
        "вторник": 1, "вт": 1,
        "среда": 2, "ср": 2,
        "четверг": 3, "чт": 3,
        "пятница": 4, "пт": 4,
        "суббота": 5, "сб": 5,
        "воскресенье": 6, "вс": 6,
    }
    
    # Daily
    if "каждый день" in text or "ежедневно" in text:
        return ("daily", 1, [])
    
    # Weekly with specific days
    if "каждую неделю" in text or "еженедельно" in text:
        days = []
        for name, weekday in days_ru.items():
            if name in text:
                days.append(weekday)
        return ("weekly", 1, days if days else [0])
    
    # Every N weeks
    match = re.search(r'каждые\s+(\d+)\s+недели?', text)
    if match:
        interval = int(match.group(1))
        return ("weekly", interval, [])
    
    # Monthly
    if "каждый месяц" in text or "ежемесячно" in text:
        return ("monthly", 1, [])
    
    # Yearly
    if "каждый год" in text or "ежегодно" in text:
        return ("yearly", 1, [])
    
    return ("none", 1, [])


def extract_event_info(text: str) -> dict:
    """Extract event information from natural language."""
    result = {
        "title": None,
        "date": None,
        "time": None,
        "duration": None,
        "location": None,
        "category": None,
    }
    
    # Try to extract date
    dt = parse_natural_date(text)
    if dt:
        result["date"] = dt.date()
        result["time"] = dt.strftime("%H:%M") if dt.hour != 0 or dt.minute != 0 else None
    
    # Extract time separately if not found in date
    if not result["time"]:
        result["time"] = parse_natural_time(text)
    
    # Extract duration
    result["duration"] = parse_duration(text)
    
    # Extract location (after "в", "на", "@" )
    loc_match = re.search(r'[ввна]\s+([А-Яа-яA-Za-z0-9\s\-]+?)(?:\s*[,\.]|$)', text)
    if loc_match:
        result["location"] = loc_match.group(1).strip()
    
    # Simple category detection
    categories = {
        "работа": "work", "офис": "work", "встреча": "work", "совещание": "work",
        "врач": "health", "больница": "health", "тренировка": "health", "спорт": "health",
        "семья": "family", "мама": "family", "папа": "family", "дети": "family",
        "учеба": "study", "курс": "study", "экзамен": "study",
        "отдых": "leisure", "кино": "leisure", "ресторан": "leisure",
        "финансы": "finance", "банк": "finance", "оплата": "finance",
    }
    
    for keyword, category in categories.items():
        if keyword in text.lower():
            result["category"] = category
            break
    
    # Title: remove common phrases and keep the rest
    title = text
    for phrase in ["встреча", "созвон", "мероприятие", "событие"]:
        title = title.replace(phrase, "")
    title = re.sub(r'\s+', ' ', title).strip()
    result["title"] = title if title else "Событие"
    
    return result
