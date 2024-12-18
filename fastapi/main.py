from fastapi import FastAPI

from pydantic import BaseModel
from transformers import T5Tokenizer, T5ForConditionalGeneration
tokenizer = T5Tokenizer.from_pretrained("google/flan-t5-base")
model = T5ForConditionalGeneration.from_pretrained("google/flan-t5-base", device_map="auto")

app = FastAPI()

class Question(BaseModel):
    question: str

@app.get("/")
async def read_root():
    return {"Hello": "World"}

@app.post("/ask")
async def get_answer(input_text: Question):
    input_text = input_text.model_dump()
    input_text = input_text['question']
    input_ids = tokenizer(input_text, return_tensors="pt").input_ids.to("cpu")
    outputs = model.generate(input_ids)
    answer = tokenizer.decode(outputs[0])
    result = {'message': str(answer)}
    return result