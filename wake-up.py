from BingImageCreator import ImageGen
from pathlib import Path
import requests
import telebot
import os
import datetime
import pytz
from dashscope import ImageSynthesis
from telebot.types import InputMediaPhoto

bot = telebot.TeleBot(os.getenv("TG_TOKEN"))


def genertor_image_by_tongyi_wanxiang(prompt:str, image_dir:str="tmp"):
    image_dir_path = Path(image_dir)
    if not image_dir_path.exists():
        image_dir_path.mkdir()
    resp = ImageSynthesis.call(model=ImageSynthesis.Models.wanx_v1,
                            prompt=prompt,
                            n=1,
                            size='1024*1024')
    if resp.status_code != 200:
        return open(image_dir_path.joinpath(f'default.jpeg'), "rb")
    else:
        result = resp.output.results[0]
        return requests.get(result.url).content
    

def genertor_image_by_bing_creator(prompt:str,  image_dir:str="tmp"):
    image_dir_path = Path(image_dir)
    if not image_dir_path.exists():
        image_dir_path.mkdir()
    
    cookie = os.getenv("BING_TOKEN")
    image_gen = ImageGen(cookie)
    images = image_gen.get_images(prompt)
    images = [InputMediaPhoto(image) for image in images]
    return images


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
        resp_json = resp.json()["results"][0]["now"]
        return f'今天天气:{resp_json["text"]},温度:{resp_json["temperature"]}度'
    else:
        return "今天天气:晴"

def send_message_to_channel():
    time_zone = pytz.timezone(os.getenv("TIME_ZONE")) 
    wake_up_time = f"今日起床时间:{datetime.datetime.now(time_zone).strftime('%Y-%m-%d %H:%M:%S')}"
    poem = get_poem()
    weather = get_weather()
    try:
        image_or_images =  genertor_image_by_bing_creator(poem)
    except:
        # 通义万象
        image_or_images = genertor_image_by_tongyi_wanxiang(poem)
    
    if isinstance(image_or_images, list):
        image_or_images[0].caption = f'{wake_up_time}\n\n{weather}\n\n今日诗词:{poem}'
        bot.send_media_group(chat_id=os.getenv("CHAT_ID"), media=image_or_images)
    else:
        bot.send_photo(chat_id=os.getenv("CHAT_ID"), photo=image_or_images, caption=f'{wake_up_time}\n\n{weather}\n\n今日诗词:{poem}')


if __name__ == '__main__':
    send_message_to_channel()
