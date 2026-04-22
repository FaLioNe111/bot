from typing import List, Optional


def format_event_display(event: dict, lang: str = "ru") -> str:
    """Format event for display in Telegram."""
    emojis = {
        "work": "💼",
        "personal": "👤",
        "health": "🏃",
        "family": "👨‍👩‍👧",
        "study": "🎓",
        "leisure": "🎉",
        "finance": "💰",
    }
    
    category_emoji = emojis.get(event.get("category", "personal"), "📅")
    
    lines = [f"{category_emoji} *{event['title']}*"]
    
    # Date and time - handle both string and date objects
    event_date = event["date"]
    if isinstance(event_date, str):
        from datetime import datetime
        event_date = datetime.fromisoformat(event_date).date()
    
    date_str = event_date.strftime("%a, %d %B" if lang == "ru" else "%a, %B %d")
    if event.get("time"):
        end_time = ""
        if event.get("duration_min"):
            from datetime import datetime, timedelta
            start = datetime.strptime(event["time"], "%H:%M")
            end = start + timedelta(minutes=event["duration_min"])
            end_time = f"–{end.strftime('%H:%M')}"
        lines.append(f"📆 {date_str} · {event['time']}{end_time}")
    else:
        lines.append(f"📆 {date_str} (весь день)")
    
    # Location
    if event.get("location"):
        lines.append(f"📍 {event['location']}")
    
    # Category tag
    category_names_ru = {
        "work": "Работа",
        "personal": "Личное",
        "health": "Здоровье",
        "family": "Семья",
        "study": "Обучение",
        "leisure": "Досуг",
        "finance": "Финансы",
    }
    if event.get("category"):
        cat_name = category_names_ru.get(event["category"], event["category"])
        lines.append(f"🏷️ {cat_name}")
    
    # Description
    if event.get("description"):
        lines.append(f"📝 {event['description']}")
    
    return "\n".join(lines)


def format_day_schedule(events: List[dict], date, lang: str = "ru") -> str:
    """Format full day schedule."""
    day_names_ru = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"]
    month_names_ru = ["января", "февраля", "марта", "апреля", "мая", "июня",
                      "июля", "августа", "сентября", "октября", "ноября", "декабря"]
    
    if lang == "ru":
        day_name = day_names_ru[date.weekday()]
        month_name = month_names_ru[date.month - 1]
        header = f"📅 *{day_name}, {date.day} {month_name}*"
    else:
        header = f"📅 *{date.strftime('%A, %B %d')}*"
    
    lines = [header, ""]
    
    if not events:
        lines.append("Нет событий на этот день")
    else:
        for event in sorted(events, key=lambda x: x.get("time") or "00:00"):
            time_str = event.get("time") or "—"
            duration = ""
            if event.get("duration_min"):
                hours = event["duration_min"] // 60
                mins = event["duration_min"] % 60
                if hours:
                    duration = f" · {hours}ч" + (f" {mins}мин" if mins else "")
                else:
                    duration = f" · {mins}мин"
            
            emoji = "📅"
            if event.get("category"):
                emoji_map = {"work": "💼", "health": "🏃", "personal": "👤"}
                emoji = emoji_map.get(event["category"], "📅")
            
            lines.append(f"{time_str} {emoji} {event['title']}{duration}")
    
    lines.append("")
    lines.append(f"Всего событий: {len(events)}")
    
    return "\n".join(lines)


def format_note_display(note: dict, lang: str = "ru") -> str:
    """Format note for display."""
    lines = [f"📝 *{note['title']}*"]
    
    if note.get("tags"):
        lines.append(f"🏷️ {', '.join(note['tags'])}")
    
    if note.get("folder"):
        lines.append(f"📁 {note['folder']}")
    
    if note.get("pinned"):
        lines.append("📌 Закреплено")
    
    lines.append("")
    
    # Body
    if note["type"] == "checklist" and isinstance(note["body"], list):
        for item in note["body"]:
            checked = "✅" if item.get("done") else "☐"
            strike = "~~" if item.get("done") else ""
            lines.append(f"{checked} {strike}{item['text']}{strike}")
    else:
        lines.append(note["body"])
    
    # Updated
    updated = note.get("updated_at")
    if updated:
        if isinstance(updated, str):
            from datetime import datetime
            updated = datetime.fromisoformat(updated)
        lines.append("")
        lines.append(f"Обновлено: {updated.strftime('%d.%m.%Y, %H:%M')}")
    
    return "\n".join(lines)


def format_habit_stats(habits: List[dict], lang: str = "ru") -> str:
    """Format habit tracker display."""
    lines = ["🔁 *Мои привычки*", ""]
    
    for habit in habits:
        icon = habit.get("icon", "🎯")
        name = habit["name"]
        streak = habit.get("current_streak", 0)
        best = habit.get("best_streak", 0)
        
        # Show last 7 days
        completions = habit.get("completions", [])
        from datetime import datetime, timedelta
        today = datetime.now().date()
        
        week_display = ""
        for i in range(6, -1, -1):
            day = today - timedelta(days=i)
            done = day in completions
            week_display += "✅" if done else "☐"
        
        lines.append(f"{icon} {name:20} {week_display}  🔥 {streak} дней")
    
    return "\n".join(lines)


def format_buttons_inline(buttons_config: list) -> str:
    """Format button configuration for backend processing."""
    output = ["[BUTTONS]"]
    for row in buttons_config:
        row_str = "row: " + " ".join([f"[{text} | {callback}]" for text, callback in row])
        output.append(row_str)
    output.append("[/BUTTONS]")
    return "\n".join(output)


def format_reply_keyboard(buttons: List[List[str]]) -> str:
    """Format reply keyboard configuration."""
    output = ["[REPLY_KEYBOARD]"]
    for row in buttons:
        output.append("[" + "] [".join(row) + "]")
    output.append("[/REPLY_KEYBOARD]")
    return "\n".join(output)
