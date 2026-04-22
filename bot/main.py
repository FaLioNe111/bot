from aiogram import Bot, Dispatcher, types
from aiogram.filters import Command
from aiogram.types import ReplyKeyboardMarkup, KeyboardButton, InlineKeyboardMarkup, InlineKeyboardButton
import asyncio
from datetime import datetime

from config.settings import settings
from services.storage import storage
from services.core_services import calendar_service, note_service, reminder_service, habit_service
from utils.formatters import (
    format_event_display, format_day_schedule, format_note_display,
    format_habit_stats, format_buttons_inline, format_reply_keyboard
)


# Initialize bot (lazy - only when token is set)
bot = None
dp = Dispatcher()


def get_bot():
    """Get bot instance, initializing if needed."""
    global bot
    if bot is None and settings.BOT_TOKEN:
        bot = Bot(token=settings.BOT_TOKEN)
    return bot


def get_main_menu_keyboard() -> ReplyKeyboardMarkup:
    """Get main menu reply keyboard."""
    keyboard = [
        ["📅 Сегодня", "📆 Неделя", "📝 Заметки"],
        ["⏰ Напомнить", "✅ Задачи", "⚙️ Настройки"],
    ]
    return ReplyKeyboardMarkup(
        keyboard=[[KeyboardButton(text=btn) for btn in row] for row in keyboard],
        resize_keyboard=True
    )


@dp.message(Command("start"))
async def cmd_start(message: types.Message):
    """Handle /start command - onboarding."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    
    if not user:
        # Create new user
        user = storage.create_user(
            user_id=None,
            telegram_id=message.from_user.id,
            name=message.from_user.first_name or "User"
        )
        
        welcome_text = (
            f"👋 Привет! Я Ora — твой помощник по делам.\n\n"
            f"Умею:\n"
            f"📅 Вести календарь\n"
            f"📝 Хранить заметки\n"
            f"⏰ Напоминать\n"
            f"✅ Следить за привычками\n\n"
            f"Начнём? Скажи мне что-нибудь вроде:\n"
            f"• \"встреча завтра в 10\"\n"
            f"• \"напомни купить хлеб через 2 часа\"\n"
            f"• \"запиши идею...\""
        )
        
        buttons = format_buttons_inline([
            [("📅 Добавить событие", "onboard_event"), ("📝 Создать заметку", "onboard_note")],
            [("⏰ Поставить напоминание", "onboard_remind"), ("🎯 Настроить привычку", "onboard_habit")],
            [("⚙️ Настройки", "settings")],
        ])
        
        await message.answer(welcome_text, parse_mode="Markdown", reply_markup=get_main_menu_keyboard())
        await message.answer(buttons)
    else:
        await message.answer(f"👋 С возвращением, {user['name']}!", reply_markup=get_main_menu_keyboard())


@dp.message(Command("today"))
async def cmd_today(message: types.Message):
    """Show today's schedule."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    from datetime import datetime
    today = datetime.now()
    events = calendar_service.get_day_schedule(user["id"], today)
    
    schedule = format_day_schedule(events, today.date(), user.get("language", "ru"))
    
    buttons = format_buttons_inline([
        [("+ Добавить событие", "add_event_today"), ("← Вчера", "day_prev"), ("Завтра →", "day_next")],
    ])
    
    await message.answer(schedule, parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("week"))
async def cmd_week(message: types.Message):
    """Show week schedule."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    from datetime import datetime
    schedule = calendar_service.get_week_schedule(user["id"])
    
    lines = ["📆 *Недельный обзор*", ""]
    
    day_names_ru = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"]
    month_names_ru = ["янв", "фев", "мар", "апр", "мая", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"]
    
    for date_str, events in sorted(schedule.items()):
        dt = datetime.fromisoformat(date_str)
        day_name = day_names_ru[dt.weekday()]
        month_name = month_names_ru[dt.month - 1]
        
        event_count = len(events)
        lines.append(f"*{day_name}, {dt.day} {month_name}* — {event_count} событий")
        
        for event in events[:3]:  # Show max 3 per day
            time_str = event.get("time") or "—"
            lines.append(f"  {time_str} · {event['title']}")
        
        if event_count > 3:
            lines.append(f"  ... и ещё {event_count - 3}")
        
        lines.append("")
    
    buttons = format_buttons_inline([
        [("📅 К сегодня", "day_today")],
    ])
    
    await message.answer("\n".join(lines), parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("notes"))
async def cmd_notes(message: types.Message):
    """Show notes list."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    notes = storage.get_user_notes(user["id"])
    
    if not notes:
        await message.answer("📝 У тебя пока нет заметок.\n\nНапиши мне что-то вроде:\n\"запиши идею для проекта\"")
        return
    
    lines = ["📝 *Заметки*", ""]
    
    # Show pinned first
    pinned = [n for n in notes if n.get("pinned")]
    others = [n for n in notes if not n.get("pinned")]
    
    for note in pinned[:5]:
        pin_emoji = "📌 " if note.get("pinned") else ""
        lines.append(f"{pin_emoji}{note['title']}")
    
    if others:
        lines.append("")
        for note in others[:5]:
            lines.append(note['title'])
    
    total = len(notes)
    if total > 10:
        lines.append(f"\n... и ещё {total - 10}")
    
    buttons = format_buttons_inline([
        [("➕ Новая заметка", "add_note")],
    ])
    
    await message.answer("\n".join(lines), parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("habits"))
async def cmd_habits(message: types.Message):
    """Show habits tracker."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    habits = storage.get_user_habits(user["id"])
    
    if not habits:
        await message.answer("🔁 У тебя пока нет привычек.\n\nНапиши:\n\"хочу отслеживать привычку — читать 20 минут каждый день\"")
        return
    
    stats = format_habit_stats(habits, user.get("language", "ru"))
    
    buttons = format_buttons_inline([
        [("➕ Добавить привычку", "add_habit")],
    ])
    
    await message.answer(stats, parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("settings"))
async def cmd_settings(message: types.Message):
    """Show settings."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    plan_emoji = "🚀" if user.get("plan") == "pro" else "🆓"
    plan_name = "Pro" if user.get("plan") == "pro" else "Free"
    
    settings_text = (
        f"⚙️ *Настройки*\n\n"
        f"🌍 Часовой пояс: {user.get('timezone', 'Europe/Moscow')}\n"
        f"🌐 Язык: {user.get('language', 'ru').upper()}\n"
        f"📦 План: {plan_emoji} {plan_name}\n"
    )
    
    buttons = format_buttons_inline([
        [("🌍 Часовой пояс", "set_timezone"), ("🌐 Язык", "set_language")],
        [("🚀 Обновить до Pro", "upgrade")],
    ])
    
    await message.answer(settings_text, parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("help"))
async def cmd_help(message: types.Message):
    """Show help."""
    help_text = (
        "📖 *Справка*\n\n"
        "*Команды:*\n"
        "/start — Главное меню\n"
        "/today — Расписание на сегодня\n"
        "/week — Недельный обзор\n"
        "/notes — Список заметок\n"
        "/habits — Трекер привычек\n"
        "/settings — Настройки\n"
        "/help — Эта справка\n\n"
        "*Примеры запросов:*\n"
        "• встреча с Алексом в пятницу в 15:00\n"
        "• напомни купить хлеб через 2 часа\n"
        "• запиши идею для проекта\n"
        "• хочу отслеживать привычку — бег по утрам"
    )
    
    await message.answer(help_text, parse_mode="Markdown")


@dp.message(Command("upgrade"))
async def cmd_upgrade(message: types.Message):
    """Show upgrade info."""
    upgrade_text = (
        "🚀 *Ora Pro*\n\n"
        "*Бесплатно навсегда:*\n"
        "• До 50 событий/мес\n"
        "• До 30 заметок\n"
        "• До 15 напоминаний\n"
        "• До 3 привычек\n\n"
        "*Pro (~2$/мес):*\n"
        "• Всё без ограничений ✅\n"
        "• Утренний брифинг\n"
        "• Полная статистика\n"
        "• Расширенный поиск\n"
        "• Интеграция с Google Calendar\n"
        "• Без рекламы\n\n"
        "Первый месяц бесплатно!"
    )
    
    buttons = format_buttons_inline([
        [("🚀 Попробовать Pro бесплатно 7 дней", "upgrade_trial")],
        [("❌ Не сейчас", "dismiss")],
    ])
    
    await message.answer(upgrade_text, parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("tasks"))
async def cmd_tasks(message: types.Message):
    """Show tasks for today."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    # Get checklist notes as tasks
    notes = storage.get_user_notes(user["id"])
    tasks = []
    
    for note in notes:
        if note.get("type") == "checklist" and isinstance(note.get("body"), list):
            for item in note["body"]:
                if not item.get("done"):
                    tasks.append({
                        "text": item["text"],
                        "note_title": note["title"],
                        "note_id": note["id"]
                    })
    
    if not tasks:
        await message.answer("✅ Все задачи выполнены!\\n\\nНет активных задач на сегодня.")
        return
    
    lines = ["✅ *Задачи на сегодня*", ""]
    for i, task in enumerate(tasks[:10], 1):
        lines.append(f"☐ {task['text']} ({task['note_title']})")
    
    if len(tasks) > 10:
        lines.append(f"... и ещё {len(tasks) - 10}")
    
    completed = sum(1 for n in notes if n.get("type") == "checklist" 
                   for item in n.get("body", []) if item.get("done"))
    total = completed + len(tasks)
    progress = int(completed / total * 100) if total else 0
    
    lines.append("")
    lines.append(f"Выполнено: {completed}/{total} · Прогресс: {progress}%")
    
    buttons = format_buttons_inline([
        [("➕ Добавить задачу", "add_task")],
    ])
    
    await message.answer("\\n".join(lines), parse_mode="Markdown")
    await message.answer(buttons)


@dp.message(Command("search"))
async def cmd_search(message: types.Message):
    """Search across all data."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    # Extract query from command args or wait for next message
    text = message.text.strip()
    query = text.replace("/search", "").strip()
    
    if not query:
        await message.answer("🔍 Введите поисковый запрос:\\n\\nНапример:\\n• найди заметки про отпуск\\n• покажи события с врачом")
        return
    
    results = []
    
    # Search notes
    notes_results = note_service.search_notes(user["id"], query)
    for note in notes_results[:5]:
        results.append(f"📝 {note['title']}")
    
    # Search events by title
    events = storage.get_user_events(user["id"])
    query_lower = query.lower()
    for event in events[:5]:
        if query_lower in event.get("title", "").lower():
            date_str = event.get("date", "")
            results.append(f"📅 {event['title']} ({date_str})")
    
    if not results:
        await message.answer(f"🔍 Ничего не найдено по запросу \\\"{query}\\\"")
        return
    
    lines = [f"🔍 *Результаты поиска:*", ""]
    lines.extend(results)
    
    if len(notes_results) > 5 or len(events) > 5:
        lines.append(f"\\n... показаны первые результаты")
    
    await message.answer("\\n".join(lines), parse_mode="Markdown")


@dp.message(Command("export"))
async def cmd_export(message: types.Message):
    """Export user data (PRO feature)."""
    user = storage.get_user_by_telegram_id(message.from_user.id)
    if not user:
        await message.answer("Сначала нажмите /start")
        return
    
    if user.get("plan") != "pro":
        upgrade_text = (
            "📤 *Экспорт данных*\\n\\n"
            "Эта функция доступна только в Pro-плане.\\n\\n"
            "Pro включает:\\n"
            "• Экспорт всех данных (JSON)\\n"
            "• Полная статистика\\n"
            "• Интеграция с Google Calendar\\n"
            "• Без ограничений"
        )
        buttons = format_buttons_inline([
            [("🚀 Попробовать Pro бесплатно 7 дней", "upgrade_trial")],
            [("❌ Не сейчас", "dismiss")],
        ])
        await message.answer(upgrade_text, parse_mode="Markdown")
        await message.answer(buttons)
        return
    
    # Generate export
    import json
    
    export_data = {
        "user": user,
        "events": storage.get_user_events(user["id"]),
        "notes": storage.get_user_notes(user["id"]),
        "habits": storage.get_user_habits(user["id"]),
        "reminders": storage.get_pending_reminders(user["id"]),
        "exported_at": datetime.now().isoformat(),
    }
    
    # Convert to JSON string
    json_str = json.dumps(export_data, ensure_ascii=False, indent=2, default=str)
    
    # Send as file
    from io import BytesIO
    file = BytesIO(json_str.encode('utf-8'))
    file.name = f"ora_export_{datetime.now().strftime('%Y%m%d')}.json"
    
    await message.answer_document(file, caption="📤 Ваши данные экспортированы")


@dp.callback_query(lambda c: c.data.startswith("confirm_"))
async def callback_confirm(callback_query: types.CallbackQuery):
    """Handle confirmation callbacks."""
    action = callback_query.data.split("_", 1)[1]
    
    if action == "create":
        # Confirm event creation despite conflict
        await callback_query.message.edit_text("✅ Событие создано")
        # In real implementation, would create the event here
    elif action == "upgrade_trial":
        # Start Pro trial
        user = storage.get_user_by_telegram_id(callback_query.from_user.id)
        if user:
            from datetime import timedelta
            storage.update_user(
                user["id"],
                plan="trial",
                trial_ends_at=datetime.now() + timedelta(days=7)
            )
        await callback_query.message.edit_text("🚀 Pro-план активирован на 7 дней!")
    
    await callback_query.answer()


@dp.callback_query(lambda c: c.data.startswith("remind_"))
async def callback_reminder_setup(callback_query: types.CallbackQuery):
    """Handle reminder setup for events."""
    parts = callback_query.data.split("_")
    minutes = int(parts[1])
    event_id = parts[2]
    
    user = storage.get_user_by_telegram_id(callback_query.from_user.id)
    event = storage.get_event(event_id)
    
    if event:
        from datetime import datetime, timedelta
        event_dt = datetime.fromisoformat(f"{event['date']}T{event['time']}")
        remind_time = event_dt - timedelta(minutes=minutes)
        
        reminder, _ = reminder_service.create_reminder(
            user["id"],
            f"Напоминание: {event['title']}",
            remind_time,
            target_type="event",
            target_id=event_id
        )
        
        await callback_query.message.edit_text(f"⏰ Напоминание установлено на {remind_time.strftime('%d.%m %H:%M')}")
    
    await callback_query.answer()


@dp.callback_query(lambda c: c.data.startswith("pin_note_"))
async def callback_pin_note(callback_query: types.CallbackQuery):
    """Handle note pinning."""
    note_id = callback_query.data.split("_", 2)[2]
    
    note = storage.get_note(note_id)
    if note:
        storage.update_note(note_id, pinned=not note.get("pinned", False))
        status = "📌 Закреплено" if not note.get("pinned") else "📍 Откреплено"
        await callback_query.message.edit_text(f"{status}: {note['title']}")
    
    await callback_query.answer()


@dp.callback_query(lambda c: c.data == "upgrade_trial")
async def callback_upgrade_trial(callback_query: types.CallbackQuery):
    """Handle Pro trial upgrade."""
    user = storage.get_user_by_telegram_id(callback_query.from_user.id)
    
    if user:
        from datetime import timedelta
        storage.update_user(
            user["id"],
            plan="trial",
            trial_ends_at=datetime.now() + timedelta(days=7)
        )
        await callback_query.message.edit_text(
            "🚀 *Pro активирован!*\\n\\n"
            "Следующие 7 дней вам доступны:\\n"
            "• Безлимитные события и заметки\\n"
            "• Утренний брифинг\\n"
            "• Полная статистика привычек\\n"
            "• Расширенный поиск\\n"
            "• Экспорт данных\\n\\n"
            "По окончании trials — 2$/мес."
        )
    
    await callback_query.answer()


@dp.callback_query(lambda c: c.data == "dismiss")
async def callback_dismiss(callback_query: types.CallbackQuery):
    """Handle dismiss/cancel button."""
    await callback_query.message.delete()
    await callback_query.answer()


@dp.callback_query(lambda c: c.data.startswith("snooze_"))
async def callback_snooze(callback_query: types.CallbackQuery):
    """Handle reminder snooze."""
    parts = callback_query.data.split("_")
    minutes = int(parts[1])
    reminder_id = parts[2]
    
    reminder_service.snooze_reminder(reminder_id, minutes)
    
    await callback_query.message.edit_text(f"⏰ Напоминание отложено на {minutes} мин")
    await callback_query.answer()


async def send_morning_briefing():
    """Send morning briefing to users who enabled it."""
    from datetime import datetime
    
    users_with_briefing = [u for u in storage.users.values() if u.get("briefing_time")]
    
    current_time = datetime.now().strftime("%H:%M")
    
    for user in users_with_briefing:
        if user.get("briefing_time") != current_time:
            continue
        
        # Check if PRO or show limited version
        is_pro = user.get("plan") in ["pro", "trial"]
        
        today_events = calendar_service.get_day_schedule(user["id"], datetime.now())
        
        greeting = "☀️ Доброе утро" if is_pro else "🌞 Привет"
        
        lines = [
            f"{greeting}, {user['name']}!",
            f"📅 {datetime.now().strftime('%A, %d %B')}",
            "",
        ]
        
        if today_events:
            lines.append("Сегодня у тебя:")
            for event in today_events[:5]:
                time_str = event.get("time") or "—"
                lines.append(f"• {time_str} — {event['title']}")
        else:
            lines.append("Сегодня свободный день 🎉")
        
        # Add habits reminder
        habits = storage.get_user_habits(user["id"])
        if habits:
            lines.append("")
            lines.append("Привычки:")
            for habit in habits[:3]:
                icon = habit.get("icon", "🎯")
                lines.append(f"• {icon} {habit['name']}")
        
        # Add tip/ad for free users
        if not is_pro and user.get("ads_enabled"):
            lines.append("")
            tips = [
                "💡 *Совет дня:* Начни с самого сложного дела — это даст энергию на остаток дня.",
                "💡 *Совет дня:* Делай перерывы каждые 90 минут для максимальной продуктивности.",
                "💡 *Совет дня:* Планируй завтрашний день с вечера — так ты сэкономишь время утром.",
            ]
            import random
            lines.append(random.choice(tips))
            
            # Partner ad example
            lines.append("")
            lines.append("📢 Партнёрский совет: Попробуй Notion для организации рабочих проектов — бесплатно для личного использования.")
        
        text = "\n".join(lines)
        
        # In real implementation, would send via bot
        print(f"Briefing for {user['telegram_id']}: {text}")


async def check_reminders_loop():
    """Background task to check and send reminders."""
    while True:
        now = datetime.now()
        
        for user_id in storage.user_reminders.keys():
            reminders = storage.get_pending_reminders(user_id)
            
            for reminder in reminders:
                trigger_time = datetime.fromisoformat(reminder["trigger_time"])
                
                if trigger_time <= now:
                    # Send reminder
                    user = storage.get_user(user_id)
                    if user:
                        # In real implementation, would send via bot
                        print(f"Reminder for {user['telegram_id']}: {reminder['message']}")
                    
                    # Update status
                    storage.update_reminder(reminder["id"], status="sent")
        
        await asyncio.sleep(60)  # Check every minute


async def main():
    """Main entry point."""
    if not settings.BOT_TOKEN:
        print("❌ BOT_TOKEN not set. Please set it in .env file")
        print("   Copy .env.example to .env and add your bot token from @BotFather")
        return
    
    # Initialize bot
    get_bot()
    
    print("🤖 Ora bot starting...")
    
    # Start background tasks
    asyncio.create_task(check_reminders_loop())
    
    # Schedule morning briefings (would use APScheduler in production)
    # For now, just run once at start for demo
    asyncio.create_task(send_morning_briefing())
    
    await dp.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main())
