FROM rayproject/ray:2.9.0

RUN pip install torch diffusers transformers xformers accelerate

WORKDIR /app

COPY . /app

CMD [ "python", "main.py", "serve" ]
