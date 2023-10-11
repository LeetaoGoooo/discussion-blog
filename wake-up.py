from BingImageCreator import ImageGen
from pathlib import Path
import requests
import telebot
import os
from random import randint
import datetime
import pytz

bot = telebot.TeleBot(os.getenv("TG_TOKEN"))

def genertor_image_by_bing_creator(cookie:str, prompt:str,  image_dir:str="tmp"):
    image_dir_path = Path(image_dir)
    if not image_dir_path.exists():
        image_dir_path.mkdir()
    try:
        image_gen = ImageGen(cookie)
        images = image_gen.get_images(prompt)
        image_gen.save_images(images, image_dir)
        image_index = randint(0, len(images)-1)
    except:
        image_index = "default"
    return open(image_dir_path.joinpath(f'{image_index}.jpeg'), "rb")


def get_poem():
    resp = requests.get("https://v2.jinrishici.com/one.json")
    if resp.status_code == 200:
        return resp.json()['data']['content']
    else:
        return "早上好"

def get_weather():
    api_key = os.getenv("WEATHER_API_KEY")
    url = f'https://api.seniverse.com/v3/weather/now.json?key={api_key}&location=shanghai&language=zh-Hans&unit=c'
    resp = requests.get(url)
    if resp.status_code == 200:
        resp_json = resp.json()
        return f'今天天气:{resp_json["text"]},温度:{resp_json["temperature"]}度'
    else:
        return "今天天气:晴"

def send_message_to_channel():
    time_zone = pytz.timezone(os.getenv("TIME_ZONE")) 
    wake_up_time = f"今日起床时间:{datetime.datetime.now(time_zone).strftime('%Y-%m-%d %H:%M:%S')}"
    poem = get_poem()
    weather = get_weather()
    cookie = os.getenv("BING_TOKEN")
    image =  genertor_image_by_bing_creator(cookie, poem)
    bot.send_photo(chat_id=os.getenv("CHAT_ID"), photo=image, caption=f'{wake_up_time}\n\n{weather}\n\n今日诗词:{poem}')


if __name__ == '__main__':
    send_message_to_channel()