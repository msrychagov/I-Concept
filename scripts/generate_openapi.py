"""Generate the embedded OpenAPI 3 document. Run from the repository root."""
import json
import re
from pathlib import Path

S = {}
def ref(name): return {'$ref': '#/components/schemas/' + name}
def obj(props, required=(), extra=False):
    result = {'type': 'object', 'properties': props, 'additionalProperties': extra}
    if required: result['required'] = list(required)
    return result
def arr(item): return {'type': 'array', 'items': item}
def string(**kw): return {'type': 'string', **kw}
def integer(**kw): return {'type': 'integer', **kw}
def number(**kw): return {'type': 'number', **kw}
def boolean(**kw): return {'type': 'boolean', **kw}
def go_type(value):
    if value.startswith('*'): return {**go_type(value[1:]), 'nullable': True}
    if value.startswith('[]'): return {**arr(go_type(value[2:])), 'nullable': True}
    if value.startswith('map[string]'): return {'type': 'object', 'additionalProperties': go_type(value[11:]), 'nullable': True}
    return {'string': string(), 'int': integer(), 'uint32': integer(minimum=0), 'float64': number(), 'bool': boolean()}.get(value, ref(value))
# Domain schemas reuse the persisted model's JSON field names.
for file in Path('internal/domain/model').glob('*.go'):
    for name, body in re.findall(r'type (\w+) struct \{(.*?)\n\}', file.read_text(), re.S):
        if name in ['Account','Login','Database','SessionRecord']: continue
        props, required = {}, []
        for typ, tag in re.findall(r'\w+\s+([^\s]+)\s+`json:"([^"]+)"`', body):
            key = tag.split(',')[0]; props[key] = go_type(typ)
            if 'omitempty' not in tag: required.append(key)
        S[name] = obj(props, required)
S['ProfileInput'] = obj(S['Profile']['properties'], ['name','author','values'])
S['ProfileInput']['description'] = 'Полный шаблон. ID при создании назначает сервер, при обновлении берётся из URL. Ориентиры выбранных показателей фиксируются сервером как 5. Если questions не передан, используется полный банк.'
S['ProfileInput']['example'] = {'name':'Короткий шаблон','author':'Иванов Иван','wave':'Входная диагностика','values':{'Q01':5}}
S['PublicAccount'] = obj({'id':string(),'name':string(),'email':string(format='email'),'platoon':string()}, ['id','name','email','platoon'])
S['AuthResponse'] = obj({'account':ref('PublicAccount')}, ['account'])
S['TeacherResponse'] = obj({'teacher':ref('PublicAccount')}, ['teacher'])
S['StudentResponse'] = obj({'student':ref('PublicAccount')}, ['student'])
S['OK'] = obj({'ok':boolean(example=True)}, ['ok'])
S['Error'] = obj({'error':string(),'conflict':boolean(),'partial':boolean()}, ['error'])
S['LoginInput'] = obj({'email':string(format='email',maxLength=254),'password':string(format='password',maxLength=256)}, ['email','password'])
S['LoginInput']['example'] = {'email':'teacher@example.test','password':'TeacherDemo2026!'}
S['RegisterInput'] = obj({'name':string(maxLength=200,description='Имя и фамилия, не менее двух слов.'),'email':string(format='email',maxLength=254),'password':string(format='password',minLength=10,maxLength=256),'platoon':string(maxLength=80,description='Обязательно для студента.')}, ['name','email','password'])
S['TeacherProfileInput'] = obj({'name':string(maxLength=200),'email':string(format='email',maxLength=254),'currentPassword':string(format='password',maxLength=256,description='Обязательно при изменении электронной почты.')}, ['name','email'])
S['PasswordInput'] = obj({'currentPassword':string(format='password',maxLength=256),'newPassword':string(format='password',minLength=10,maxLength=256),'confirmPassword':string(format='password',minLength=10,maxLength=256)}, ['currentPassword','newPassword','confirmPassword'])
S['AssignmentInput'] = obj({'title':string(maxLength=200),'profileId':string(),'platoon':string(maxLength=80,description='Пустая строка — всем взводам.'),'activity':boolean(default=False),'motivation':boolean(default=False)}, ['title','profileId'])
S['AssignmentInput']['example'] = {'title':'Входное тестирование','profileId':'ID-шаблона','platoon':'101','activity':False,'motivation':False}
S['AssignmentStatusInput'] = obj({'active':boolean()}, ['active'])
S['AssignmentStatusInput']['example'] = {'active':False}
S['AssignmentSummary'] = obj({'id':string(),'title':string(),'platoon':string(),'active':boolean(),'created':string(format='date-time'),'profileName':string()}, ['id','title','platoon','active','created','profileName'])
S['StudentAssignment'] = obj({'id':string(),'title':string(),'created':string(format='date-time'),'active':boolean(),'completed':boolean(),'status':string(),'sessionId':string(),'finishedAt':string(format='date-time'),'answered':integer(),'total':integer()}, ['id','title','created','active','completed','status'])
S['StudentAssignments'] = obj({'assignments':arr(ref('StudentAssignment'))}, ['assignments'])
S['Assignments'] = obj({'assignments':arr(ref('AssignmentSummary'))}, ['assignments'])
S['DashboardSession'] = obj({**{k:string() for k in ['id','studentId','assignmentId','name','platoon','title','updated']}, 'completed':boolean(),'answered':integer(),'total':integer()}, ['id','studentId','assignmentId','name','platoon','title','updated','completed','answered','total'])
S['Dashboard'] = obj({'students':arr(ref('PublicAccount')),'assignments':arr(ref('AssignmentSummary')),'sessions':arr(ref('DashboardSession'))}, ['students','assignments','sessions'])
S['Templates'] = obj({'profiles':{'type':'array','items':ref('Profile'),'nullable':True},'emptyProfile':ref('Profile')}, ['profiles','emptyProfile'])
S['SessionSummary'] = obj({**{k:string() for k in ['id','kind','participant','group','wave','observer','created','updated','profileName','profileId']},'parent':string(nullable=True),'completed':boolean(),'revoked':boolean(),'revision':integer(),'answered':integer(),'total':integer()})
S['Sessions'] = obj({'sessions':arr(ref('SessionSummary'))}, ['sessions'])
S['StartSession'] = obj({'url':string(example='/survey?id=ID-сессии')}, ['url'])
S['ConsentInput'] = obj({'value':boolean(enum=[True]),'revision':integer(minimum=0)}, ['value','revision'])
S['AnswerInput'] = obj({'value':integer(nullable=True,description='Значение в пределах шкалы вопроса. null удаляет ответ.'),'note':string(maxLength=4000),'revision':integer(minimum=0)}, ['value','revision'])
S['AnswerInput']['example'] = {'value':5,'note':'','revision':1}
S['CompletionInput'] = obj({'allowPartial':boolean(default=False),'revision':integer(minimum=0)}, ['revision'])
S['RevisionResponse'] = obj({'revision':integer(),'updated':string(format='date-time'),'completed':boolean()}, ['revision','updated','completed'])
S['PublicQuestion'] = obj({'id':string(),'module':string(),'text':string(),'min':integer(),'max':integer(),'type':string(),'indicator':string(),'options':arr(obj({'id':string(),'text':string()}, ['id','text']))}, ['id','module','text','min','max','type'])
S['Survey'] = obj({'session':ref('Session'),'items':arr(ref('PublicQuestion')),'modules':{'type':'object','additionalProperties':string()},'version':string()}, ['session','items','modules','version'])
S['Survey']['description'] = 'Только собственная незавершённая сессия активного назначения. В session.profile передаётся пустая структура; ключи, веса вариантов и расчёты отсутствуют.'
S['ReportRow'] = obj({'id':string(),'name':string(),'block':string(),'component':string(),'actual':number(nullable=True),'target':number(nullable=True),'gap':number(nullable=True),'status':string(),'coverage':integer(),'total':integer()}, ['id','name','block','component','actual','target','gap','status','coverage','total'])
S['Report'] = obj({'rows':arr(ref('ReportRow')),'components':arr(obj({'code':string(),'name':string(),'actual':number(nullable=True),'target':number(nullable=True),'n':integer(),'total':integer()})), **{k:arr({'type':'object','additionalProperties':True}) for k in ['tags','images','sjt','history','recommendations','motivation','activity']},'profile':ref('Profile'),'notes':{'type':'object','additionalProperties':string()},'plan':{'type':'array','items':ref('PlanItem'),'nullable':True},'answered':integer(),'total':integer()}, ['rows','components','answered','total'])
S['SessionReport'] = obj({'session':ref('Session'),'report':ref('Report'),'items':arr(ref('Item'))}, ['session','report','items'])
S['GroupReport'] = obj({'group':string(),'wave':string(),'profile':string(),'n':integer(),'rows':arr(obj({'id':string(),'name':string(),'block':string(),'target':number(nullable=True),'actual':number(nullable=True),'n':integer()}))}, ['group','wave','profile','n','rows'])
S['PlanInput'] = obj({'plan':arr(ref('PlanItem'))}, ['plan']);S['PlanInput']['properties']['plan']['maxItems']=100
S['PlanItem']['properties']['status']['enum']=['planned','doing','done']
paths={}
errors={400:'Неверные поля или JSON.',401:'Требуется вход.',403:'Недостаточно прав, назначение закрыто или недопустимый Origin.',404:'Ресурс не найден.',409:'Конфликт: занятая почта, завершённая сессия или устаревшая revision.',413:'Превышен размер запроса.',415:'Ожидается application/json.',422:'Остались пропуски; повторите с allowPartial=true.',500:'Ошибка сохранения данных.'}
security={'teacher':[{'teacherCookie':[]},{'basicAuth':[]}], 'profile':[{'teacherCookie':[]}], 'student':[{'studentCookie':[]}], 'auth':[]}
def endpoint(path,method,operation,tag,summary,response='OK',body=None,auth='teacher',status=200,description='',codes=()):
    operation_obj={'operationId':operation,'tags':[tag],'summary':summary,'security':security[auth],'responses':{str(status):{'description':'Успешно.','content':{'application/json':{'schema':ref(response) if isinstance(response,str) else response}}}}}
    if description:operation_obj['description']=description
    if body:operation_obj['requestBody']={'required':True,'content':{'application/json':{'schema':ref(body)}}}
    parameters=[]
    for param in re.findall(r'\{(\w+)\}',path):
        schema=string(enum=['teacher','student']) if param=='role' else string(minLength=1)
        parameters.append({'name':param,'in':'path','required':True,'schema':schema})
    if parameters:operation_obj['parameters']=parameters
    default_codes={500}
    if auth!='auth':default_codes.add(401)
    if method!='get':default_codes.add(403)
    if body:default_codes|={400,413,415}
    for code in sorted(default_codes|set(codes)):operation_obj['responses'][str(code)]={'description':errors[code],'content':{'application/json':{'schema':ref('Error')}}}
    paths.setdefault(path,{})[method]=operation_obj
    return operation_obj
for action,title,body,status in [('register','Регистрация','RegisterInput',201),('login','Вход','LoginInput',200)]:
    op=endpoint('/auth/{role}/'+action,'post',action,'Авторизация',title,'AuthResponse',body,'auth',status,'Вход создаёт HttpOnly cookie vuc_teacher или vuc_student на 24 часа. Регистрация сразу выполняет вход. role=student требует platoon.',[401,404,409])
    op['responses'][str(status)]['headers']={'Set-Cookie':{'description':'HttpOnly cookie; Secure при HTTPS; SameSite=Lax.','schema':string()}}
endpoint('/auth/{role}/logout','post','logout','Авторизация','Выход',auth='auth',description='Удаляет текущий сеанс выбранной роли и очищает cookie. Тело не требуется.',codes=[404])
endpoint('/teachers/me','get','getTeacherProfile','Профиль','Профиль преподавателя','TeacherResponse',auth='profile')
endpoint('/teachers/me','patch','updateTeacherProfile','Профиль','Изменить ФИО и почту','TeacherResponse','TeacherProfileInput','profile',description='Оба поля name и email обязательны. Смена почты требует currentPassword.',codes=[409])
endpoint('/teachers/me/password','put','changeTeacherPassword','Профиль','Сменить пароль','TeacherResponse','PasswordInput','profile',description='Сохраняет вход на текущем устройстве, завершает остальные сеансы преподавателя.')
endpoint('/students/me','get','getStudentProfile','Студент','Профиль студента','StudentResponse',auth='student')
endpoint('/students/me/assignments','get','getStudentAssignments','Студент','Назначения студента','StudentAssignments',auth='student',description='Включает доступные и закрытые назначения. Исходные ответы и расчёты не возвращаются.')
endpoint('/students/me/assignments/{id}/session','post','startAssignment','Студент','Начать или продолжить тестирование','StartSession',auth='student',description='Без тела. Возвращает URL новой или существующей незавершённой сессии. Повторный запрос не создаёт новую попытку.',codes=[400,403,409])
endpoint('/students/me/sessions/{id}','get','getSurvey','Анкета','Получить свою анкету','Survey',auth='student',codes=[403])
endpoint('/students/me/sessions/{id}/consent','put','saveConsent','Анкета','Подтвердить согласие','RevisionResponse','ConsentInput','student',codes=[409])
endpoint('/students/me/sessions/{id}/answers/{itemId}','put','saveAnswer','Анкета','Сохранить ответ и комментарий','RevisionResponse','AnswerInput','student',description='Записывает ответ и комментарий, увеличивает revision. Передайте value=null, чтобы удалить ответ. Отсутствующий note очищает комментарий.',codes=[409])
endpoint('/students/me/sessions/{id}/completion','post','completeSurvey','Анкета','Завершить тестирование','RevisionResponse','CompletionInput','student',description='Без allowPartial=true пропуски приводят к 422 с partial=true. После завершения студент не может изменять ответы.',codes=[409,422])
endpoint('/question-bank','get','getQuestionBank','Методика','Банк вопросов и показатели','Bank')
endpoint('/dashboard','get','getDashboard','Преподаватель','Данные панели преподавателя','Dashboard')
endpoint('/templates','get','listTemplates','Шаблоны','Список шаблонов и основа нового шаблона','Templates')
endpoint('/templates/{id}','get','getTemplate','Шаблоны','Получить шаблон','Profile',codes=[404])
endpoint('/templates','post','createTemplate','Шаблоны','Создать шаблон','Profile','ProfileInput',status=201)
paths['/templates']['post']['responses']['201']['headers']={'Location':{'description':'URL созданного шаблона.','schema':string()}}
endpoint('/templates/{id}','put','updateTemplate','Шаблоны','Сохранить шаблон целиком','Profile','ProfileInput',description='ID в URL имеет приоритет. Существующие назначения и сессии сохраняют прежний снимок шаблона.',codes=[404])
endpoint('/assignments','get','listAssignments','Назначения','Список назначений','Assignments')
endpoint('/assignments','post','createAssignment','Назначения','Назначить тестирование','OK','AssignmentInput',status=201)
endpoint('/assignments/{id}','patch','setAssignmentStatus','Назначения','Открыть или закрыть доступ','OK','AssignmentStatusInput',description='active=false блокирует начало и продолжение. Сохранённые ответы остаются. active=true возобновляет доступ.',codes=[404])
endpoint('/sessions','get','listSessions','Результаты','Список сессий','Sessions')
endpoint('/sessions/{id}/report','get','getSessionReport','Результаты','Результаты и ответы','SessionReport',codes=[404])
endpoint('/sessions/{id}/export','get','exportSession','Экспорт','Экспорт сессии в JSON','Session',codes=[404])
endpoint('/sessions/import','post','importSession','Экспорт','Импорт сессии','OK','Session',status=201,description='Принимает JSON экспорта. bankVersion должен быть известной версией банка, ID должен быть уникальным. Импортированная сессия помечается revoked.',codes=[409])
endpoint('/sessions/{id}/reopen','post','reopenSession','Результаты','Открыть сессию для исправления',description='Без тела. Сбрасывает completed, увеличивает revision. Назначение также должно быть активно.',codes=[404])
endpoint('/sessions/{id}/revoke','post','revokeSession','Результаты','Пометить сессию как отозванную',description='Без тела. Устанавливает исторический флаг revoked; доступ к назначению управляется PATCH /assignments/{id}.',codes=[404])
endpoint('/sessions/{id}/plan','put','updateSessionPlan','Результаты','Сохранить план сопровождения','OK','PlanInput',codes=[404])
endpoint('/reports/groups','get','getGroupReport','Результаты','Агрегированный отчёт по взводам',arr(ref('GroupReport')))
endpoint('/exports/answers.csv','get','exportAnswersCSV','Экспорт','Экспорт ответов в CSV')
paths['/exports/answers.csv']['get']['responses']['200']={'description':'CSV с ответами.','headers':{'Content-Disposition':{'schema':string(),'description':'attachment; filename="vuc-answers.csv"'}},'content':{'text/csv':{'schema':string()}}}
for path in ['/reports/groups','/exports/answers.csv']:
    paths[path]['get']['parameters']=[{'name':'group','in':'query','required':False,'description':'Номер взвода; без параметра — все взводы.','schema':string()}]
endpoint('/openapi.json','get','getOpenAPI','Документация','Описание OpenAPI',{'type':'object','additionalProperties':True},auth='auth')
S['Bank']['properties']['instructions']={'type':'object','additionalProperties':string()}
S['Survey']['properties']['instructions']={'type':'object','additionalProperties':string()}
S['EvidenceChoice']=obj({'id':string(),'text':string(),'selected':string(),'interpretation':string()})
S['EvidenceTrend']=obj({'available':boolean(),'reason':string(),'previous':number(nullable=True),'delta':number(nullable=True),'previousDate':string(),'previousSessionId':string()})
S['EvidenceRow']=obj({'id':string(),'name':string(),'block':string(),'definition':string(),'limitations':string(),'selfImage':integer(nullable=True),'statements':number(nullable=True),'coverage':integer(),'total':integer(),'choices':arr(ref('EvidenceChoice')),'trend':ref('EvidenceTrend')})
S['Report']['properties']['evidence']=arr(ref('EvidenceRow'))
S['Report']['properties']['interpretationNote']=string()
spec={'openapi':'3.0.3','info':{'title':'API диагностической системы ВУЦ','version':'1.0.0','description':'Отдельные HTTP-маршруты для каждой операции. JSON-запросы требуют Content-Type: application/json; неизвестные поля отклоняются. Лимит обычных запросов 16 КиБ, шаблонов и импорта 3 МБ, плана 300 КБ.\n\nАвторизация: вызовите /auth/teacher/login или /auth/student/login (в том числе через Try it out), браузер сохранит HttpOnly cookie и отправит её в последующих запросах. В Swagger поле cookie вручную не вводится. Basic Auth доступен только для преподавательских ресурсов при явной настройке VUC_ADMIN_PASSWORD; настройки личного профиля требуют cookie преподавателя.\n\nОшибки возвращаются как {error, conflict?, partial?}. Изменения анкеты требуют актуальной revision. Старые маршруты /api/vuc, /api/account и /api/teacher удалены.'},'servers':[{'url':'/api/v1'}],'tags':[{'name':name} for name in ['Авторизация','Профиль','Студент','Анкета','Методика','Преподаватель','Шаблоны','Назначения','Результаты','Экспорт','Документация']],'paths':paths,'components':{'schemas':S,'securitySchemes':{'teacherCookie':{'type':'apiKey','in':'cookie','name':'vuc_teacher'},'studentCookie':{'type':'apiKey','in':'cookie','name':'vuc_student'},'basicAuth':{'type':'http','scheme':'basic','description':'Только если настроен VUC_ADMIN_PASSWORD.'}}}}
Path('web/static/openapi.json').write_text(json.dumps(spec,ensure_ascii=False,indent=2)+'\n')
print(f'OpenAPI: {sum(len(v) for v in paths.values())} operations, {len(S)} schemas')
