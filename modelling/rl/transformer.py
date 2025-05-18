import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.nn.utils import weight_norm
import math

import functools
import gymnasium as gym
from dataclasses import dataclass
from ray.rllib.core.columns import Columns
from ray.rllib.core.models.base import ENCODER_OUT
from ray.rllib.core.models.configs import ModelConfig
from ray.rllib.algorithms.ppo.ppo_catalog import PPOCatalog
from ray.rllib.models.utils import get_activation_fn

class PositionalEmbedding(nn.Module):
    def __init__(self, d_model, max_len=5000):
        super(PositionalEmbedding, self).__init__()
        # Compute the positional encodings once in log space.
        pe = torch.zeros(max_len, d_model).float()
        pe.require_grad = False

        position = torch.arange(0, max_len).float().unsqueeze(1)
        div_term = (torch.arange(0, d_model, 2).float()
                    * -(math.log(10000.0) / d_model)).exp()

        pe[:, 0::2] = torch.sin(position * div_term)
        pe[:, 1::2] = torch.cos(position * div_term)

        pe = pe.unsqueeze(0)
        self.register_buffer('pe', pe)

    def forward(self, x):
        return self.pe[:, :x.size(1)]
    
class DataEmbedding(nn.Module):
    def __init__(self, c_in, d_model, dropout=0.1, max_len=512, bias=True, sinusoidal=True, use_cls_token=True):
        super(DataEmbedding, self).__init__()

        self.value_embedding =  nn.Linear(c_in, d_model, bias=bias)

        self.sinusoidal = sinusoidal
        if sinusoidal:
            self.position_embedding = PositionalEmbedding(d_model=d_model, max_len=max_len)
        else:
            self.position_embedding = nn.Parameter(torch.randn(1, max_len, d_model))

        self.use_cls_token = use_cls_token
        if use_cls_token:
            self.cls_token = nn.Parameter(torch.randn(1, 1, d_model))

        self.dropout = nn.Dropout(p=dropout)

    def forward(self, x):
        B, L, _ = x.shape

        x = self.value_embedding(x)
        
        if self.use_cls_token:
            cls = self.cls_token.expand(B, 1, -1)  # [B, 1, D]
            x = torch.cat([cls, x], dim=1)         # [B, L+1, D]

        if self.sinusoidal:
            x = x + self.position_embedding(x)
        else:
            x = x + self.position_embedding[:, :x.size(1), :]  # [B, L+1, D]
        return self.dropout(x)


class TransformerEncoder(nn.Module):
    def __init__(
        self,
        c_in: int,              # 每个 token 的维度
        d_model: int = 128,        # transformer 内部隐藏层维度
        n_head: int = 4,          # 多头注意力数量
        n_layer: int = 2,         # encoder 层数
        dropout: float = 0.1,
        max_len: int = 512,         # 最大序列长度
        act: str = "gelu",          # 激活函数
        use_cls_token: bool = True,  # 是否使用 [CLS] 方式

        embedding_dropout: float = 0.1,  # embedding 层 dropout
        embedding_bias: bool = True,        # embedding 层 bias
        embedding_sinusoidal: bool = True,  # embedding 层 sinusoidal
    ):
        super().__init__()
        self.use_cls_token = use_cls_token
        self.c_in = c_in
        self.d_model = d_model
        self.n_head = n_head
        self.n_layer = n_layer
        self.dropout = dropout
        self.max_len = max_len
        self.act = act

        # 可选的 CLS token
        if use_cls_token:
            self.cls_token = nn.Parameter(torch.randn(1, 1, d_model))

        # Positional encoding（这里使用可学习的 absolute PE）
        self.embedding = DataEmbedding(
            c_in=c_in,
            d_model=d_model,
            max_len=max_len,
            dropout=embedding_dropout,
            bias=embedding_bias,
            sinusoidal=embedding_sinusoidal,
            use_cls_token=use_cls_token
        )

        # Transformer Encoder
        encoder_layer = nn.TransformerEncoderLayer(
            d_model=d_model,
            nhead=n_head,
            dim_feedforward=d_model * 4,
            dropout=dropout,
            activation=act,
            batch_first=True,  # [B, L, D]
        )
        self.encoder = nn.TransformerEncoder(encoder_layer, num_layers=n_layer)


    def forward(self, x):
        """
        x: [B, L, input_dim]
        """
        x = x[Columns.OBS]

        B, L, _ = x.shape
        
        x = self.embedding(x)  # [B, L, D]

        x = self.encoder(x)

        if self.use_cls_token:
            x = x[:, 0]  # 取 CLS token
        else:
            x = x[:, -1]  # 取最后一个 token

        return {ENCODER_OUT: x}

@dataclass
class TransformerEncoderConfig(ModelConfig):
    # input_dims: int
    d_model: int = 128
    n_head: int = 4
    n_layer: int = 2
    dropout: float = 0.1
    max_len: int = 512
    act: str = "gelu"
    use_cls_token: bool = True
    embedding_dropout: float = 0.1
    embedding_bias: bool = True
    embedding_sinusoidal: bool = True

    @property
    def output_dims(self):
        return (self.d_model,)

    def _validate(self, framework: str = "torch"):
        """Makes sure that settings are valid."""
        if framework not in ["torch"]:
            raise ValueError(
                f"Framework {framework} not supported. Only 'torch' is supported."
            )
        
        # input_dims is a tuple (seq_len, input_dim)
        if len(self.input_dims) != 2:
            raise ValueError(
                f"input should be 2D space, but got {self.input_dims}"
            )


    def build(self, framework: str = "torch"):
        self._validate(framework)

        if framework == "torch":
            return TransformerEncoder(
                c_in=self.input_dims[1],
                d_model=self.d_model,
                n_head=self.n_head,
                n_layer=self.n_layer,
                dropout=self.dropout,
                max_len=self.max_len,
                act=self.act,
                use_cls_token=self.use_cls_token,
                embedding_dropout=self.embedding_dropout,
                embedding_bias=self.embedding_bias,
                embedding_sinusoidal=self.embedding_sinusoidal
            )
        else:
            raise ValueError(f"Framework {framework} not supported.")

class TransformerEncoderCatalog(PPOCatalog):

    def _determine_components_hook(self):
        observation_space=self.observation_space
        model_config_dict=self._model_config_dict

        self._encoder_config = TransformerEncoderConfig(
            input_dims=observation_space.shape,
            d_model=model_config_dict.get("d_model", 128),
            n_head=model_config_dict.get("n_head", 4),
            n_layer=model_config_dict.get("n_layer", 2),
            dropout=model_config_dict.get("dropout", 0.1),
            max_len=model_config_dict.get("max_len", 512),
            act=model_config_dict.get("act", "gelu"),
            use_cls_token=model_config_dict.get("use_cls_token", True),
            embedding_dropout=model_config_dict.get("embedding_dropout", 0.1),
            embedding_bias=model_config_dict.get("embedding_bias", True),
            embedding_sinusoidal=model_config_dict.get("embedding_sinusoidal", True),
        )
        # Create a function that can be called when framework is known to retrieve the
        # class type for action distributions
        self._action_dist_class_fn = functools.partial(
            self._get_dist_cls_from_action_space, action_space=self.action_space
        )

        # The dimensions of the latent vector that is output by the encoder and fed
        # to the heads.
        self.latent_dims = self._encoder_config.output_dims
