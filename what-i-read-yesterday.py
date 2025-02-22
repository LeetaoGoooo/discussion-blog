from pathlib import Path
import telebot
import os
import edge_tts
import datetime
import pytz

bot = telebot.TeleBot(os.getenv("TG_TOKEN"))

import sys
from telebot.types import InputFile

VOICE = "zh-CN-XiaoxiaoNeural"

def generate_voice(text:str):
    voice_dir_path = Path("outputs")
    if not voice_dir_path.exists():
        voice_dir_path.mkdir()
    time_zone = pytz.timezone(os.getenv("TIME_ZONE","Asia/Shanghai")) 
    today_voice_path = voice_dir_path.joinpath(f"{datetime.datetime.now(time_zone).strftime('%Y-%m-%d')}.mp3")
    communicate = edge_tts.Communicate(text, VOICE)
    communicate.save_sync(today_voice_path)
    return today_voice_path


def send_message_to_channel():
    if len(sys.argv) < 2:
        sys.exit(1)

    summary = sys.argv[1]
    voice_media = generate_voice(summary)
    voice_media_file = InputFile(file=voice_media, file_name='What I Read Yesterday')
    bot.send_audio(chat_id=os.getenv("CHAT_ID"), caption='What I Read Yesterday',audio=voice_media_file, parse_mode='markdown',title='What I Read Yesterday')


if __name__ == '__main__':
    send_message_to_channel()
