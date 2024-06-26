/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { API_URL_PREFIX } from "../../config/env";
import { BotMessage, BotMessageWithDebugInfo, DebugMessage } from "../../types/api/entities/bot";

export const permalinkLinkBuilder = (id: string): string => {
  const apiUrl = process.env.REACT_APP_API_URL_PREFIX || API_URL_PREFIX;
  const isV2API = /https?:\/\/.*v2\.postgres\.ai\b/.test(apiUrl);
  return `https://${isV2API ? 'v2.' : ''}postgres.ai/chats/${id}`;
};

export const disallowedHtmlTagsForMarkdown= [
  'script',
  'style',
  'iframe',
  'form',
  'input',
  'link',
  'meta',
  'embed',
  'object',
  'applet',
  'base',
  'frame',
  'frameset',
  'audio',
  'video',
];

export const createMessageFragment = (messages: DebugMessage[]): DocumentFragment => {
  const fragment = document.createDocumentFragment();

  messages.forEach((item) => {
    const textBeforeLink = `[${item.created_at}]: `;
    const parts = item.content.split(/(https?:\/\/[^\s]+)/g);

    const messageContent = parts.map((part) => {
      if (/https?:\/\/[^\s]+/.test(part)) {
        const link = document.createElement('a');
        link.href = part;
        link.target = '_blank';
        link.textContent = part;
        return link;
      } else {
        return document.createTextNode(part);
      }
    });

    fragment.appendChild(document.createTextNode(textBeforeLink));
    messageContent.forEach((node) => fragment.appendChild(node));
    fragment.appendChild(document.createTextNode('\n'));
  });

  return fragment;
};