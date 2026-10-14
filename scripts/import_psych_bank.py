"""Import the supplied full psychological bank without diagnostic scoring.

Usage: python3 scripts/import_psych_bank.py SOURCE.json QUESTIONNAIRE.txt
The future-serviceman block in the DOCX is intentionally excluded.
"""
import json
import re
import sys
from pathlib import Path

source = json.loads(Path(sys.argv[1]).read_text())
text = Path(sys.argv[2]).read_text()
groups = [(d['code'], d['name'], d['time_window'], list(d['items'])) for d in source['domains']]
for code, name, window, items in groups:
    if code == 'SUI':
        items.extend(source['conditional_items'])
groups.extend([
    ('SOCX', 'Социальное взаимодействие: дополнительные вопросы', 'Обычный опыт / длительный период', source['extended_social_items']),
    ('FUN', 'Влияние трудностей на повседневную жизнь', 'Текущее положение', [i for i in source['shared_items'] if i['id'].startswith('FUN')]),
    ('SUPPORT', 'Поддержка и обращение за помощью', 'Текущее положение', [i for i in source['shared_items'] if not i['id'].startswith('FUN')]),
])
result = {'modules': {'PS': 'Мое самочувстсвие'}, 'instructions': {'PS': 'Ориентируйтесь на период, указанный в вопросе. Можно выбрать «Не относится ко мне», «Предпочитаю не отвечать» или оставить вопрос без ответа. Ответы описывают ваш опыт и не являются диагнозом или оценкой пригодности. Раздел не заменяет консультацию специалиста и не является каналом экстренной помощи.'}, 'indicators': [], 'items': []}
for code, name, window, items in groups:
    indicator = 'PS_' + code
    result['indicators'].append({'id': indicator, 'name': name, 'block': 'psychological', 'component': '', 'items': ['PS_' + i['id'] for i in items], 'tags': [], 'descriptive': True, 'definition': window + '. Описание ответов по данному направлению.', 'limitations': source['validation_status'] + '. Не входит в педагогические баллы и не используется для определения пригодности.', 'source': source['name'] + ' / ' + source['version']})
    for item in items:
        if item['type'] == 'FREQ4':
            options = [{'id': str(o['value']), 'text': o['label'], 'score': 0} for o in source['general_scale']['options']]
        else:
            options = [{'id': 'SKIP' if label == 'Предпочитаю не отвечать' else 'NA' if label == 'Не относится ко мне' else str(n), 'text': label, 'score': 0} for n, label in enumerate(source['response_types'][item['type']])]
        result['items'].append({'id': 'PS_' + item['id'], 'module': 'PS', 'section': 'PS', 'indicator': indicator, 'text': item['text'], 'type': 'psychological', 'min': 0, 'max': len(options)-1, 'options': options, 'responseOnly': True, 'timeWindow': item.get('time_window', window), 'source': f"{source['name']} / {source['version']} / {item['id']}"})
ids = [i['id'][3:] for i in result['items']]
text_items = dict(re.findall(r'^([A-Z]+\d+)\.\s*(.+)$', text, re.M))
assert len(ids) == len(set(ids))
assert set(ids) == set(text_items), 'JSON and text questionnaire differ'
assert all(i['text'].strip() == text_items[i['id'][3:]].strip() for i in result['items']), 'Question wording differs'
Path('internal/questionbank/psychological.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
print(f"Imported {len(ids)} unique questions into {len(groups)} descriptive indicators; JSON and TXT match.")
