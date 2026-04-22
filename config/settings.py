import os
from dotenv import load_dotenv

load_dotenv()

class Settings:
    BOT_TOKEN: str = os.getenv("BOT_TOKEN", "")
    ADMIN_IDS: list[int] = [int(x) for x in os.getenv("ADMIN_IDS", "").split(",") if x]
    
    # Timezone default
    DEFAULT_TIMEZONE: str = "Europe/Moscow"
    
    # Limits for Free plan
    FREE_LIMITS = {
        "events_per_month": 50,
        "active_notes": 30,
        "active_reminders": 15,
        "recurring_events": 5,
        "habits": 3,
    }
    
    # Pro pricing
    PRO_PRICE_STARS: int = int(os.getenv("PRO_PRICE_STARS", "150"))
    PRO_TRIAL_DAYS: int = 7
    
    # Ads
    ADS_ENABLED_FOR_FREE: bool = True

settings = Settings()
