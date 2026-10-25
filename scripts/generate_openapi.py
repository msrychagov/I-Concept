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
        if name in ['Account','Login','Invitation','Database','SessionRecord','Portrait','PortraitUsage']: continue
        props, required = {}, []
        for typ, tag in re.findall(r'\w+\s+([^\s]+)\s+`json:"([^"]+)"`', body):
            key = tag.split(',')[0]; props[key] = go_type(typ)
            if 'omitempty' not in tag: required.append(key)
        S[name] = obj(props, required)
S['ProfileInput'] = obj(S['Profile']['properties'], ['name','author','values'])
S['ProfileInput']['description'] = 'Полный шаблон. ID при создании назначает сервер, при обновлении берётся из URL. Ориентиры выбранных показателей фиксируются сервером как 5. Если questions не передан, используется полный банк.'
S['ProfileInput']['example'] = {'name':'Короткий шаблон','author':'Иванов Иван','wave':'Входная диагностика','values':{'Q01':5}}
S['PublicAccount'] = obj({'id':string(),'name':string(),'email':string(format='email'),'platoon':string(),'platoonId':string(),'vuc':string(),'disabled':boolean()}, ['id','name','email','platoon','platoonId','vuc','disabled'])
S['AuthResponse'] = obj({'account':ref('PublicAccount')}, ['account'])
S['TeacherResponse'] = obj({'teacher':ref('PublicAccount')}, ['teacher'])
S['StudentResponse'] = obj({'student':ref('PublicAccount')}, ['student'])
S['OK'] = obj({'ok':boolean(example=True)}, ['ok'])
S['Error'] = obj({'error':string(),'conflict':boolean(),'partial':boolean()}, ['error'])
S['LoginInput'] = obj({'email':string(format='email',maxLength=254),'password':string(format='password',maxLength=256)}, ['email','password'])
S['LoginInput']['example'] = {'email':'teacher@example.test','password':'TeacherDemo2026!'}
S['RegisterInput'] = obj({'invite':string(minLength=10,maxLength=512,description='Для преподавателя — одноразовый код администратора; для студента — ссылка преподавателя или её ключ.'),'name':string(maxLength=200,description='Имя и фамилия, не менее двух слов.'),'email':string(format='email',maxLength=254),'password':string(format='password',minLength=10,maxLength=256)}, ['invite','name','email','password'])
S['TeacherInvitationInput'] = obj({'email':string(format='email',maxLength=254),'vuc':string(minLength=1,maxLength=200)}, ['email','vuc'])
S['TeacherInvitation'] = obj({'code':string(example='ABCDE-FGHJK'),'role':string(enum=['teacher']),'email':string(format='email'),'vuc':string(),'expires':string(format='date-time')}, ['code','role','email','vuc','expires'])
S['StudentLinkInput'] = obj({'platoonId':string(minLength=1)}, ['platoonId'])
S['StudentRegistrationLink'] = obj({'link':string(example='/student?invite=...'),'role':string(enum=['student']),'platoon':string(),'platoonId':string(),'vuc':string(),'expires':string(format='date-time')}, ['link','role','platoon','platoonId','vuc','expires'])
S['PlatoonInput'] = obj({'name':string(minLength=1,maxLength=80)}, ['name'])
S['VUCInput'] = obj({'vuc':string(minLength=1,maxLength=200)}, ['vuc'])
S['CreateVUCInput'] = obj({'name':string(minLength=1,maxLength=200)}, ['name'])
S['VUCList'] = obj({'vucs':arr(ref('VUC'))}, ['vucs'])
S['PlatoonManagement'] = obj({'vuc':string(),'platoons':arr(ref('Platoon')),'students':arr(ref('PublicAccount'))}, ['vuc','platoons','students'])
S['InstructorInput'] = obj({'surname':string(minLength=1,maxLength=60),'name':string(minLength=1,maxLength=60),'patronymic':string(maxLength=60),'rank':string(maxLength=80,description='Свободный текст; у гражданского персонала пусто.')}, ['surname','name'])
S['InstructorInput']['example'] = {'surname':'Иванов','name':'Игорь','patronymic':'Петрович','rank':'полковник'}
S['InstructorEntry'] = {'allOf':[ref('Instructor'),obj({'updatedByName':string(description='Кто сохранил запись последним.')}, extra=True)]}
S['InstructorList'] = obj({'vuc':string(),'instructors':arr(ref('InstructorEntry'))}, ['vuc','instructors'])
S['CohortInput'] = obj({'name':string(minLength=1,maxLength=80),'year':integer(minimum=2000,maximum=2100),'platoonIds':arr(string())}, ['name','year','platoonIds'])
S['CohortInput']['example'] = {'name':'Выпуск 2027','year':2027,'platoonIds':['ID-взвода']}
S['CohortEntry'] = {'allOf':[ref('Cohort'),obj({'createdByName':string(),'mine':boolean(description='Выпуск создан вызывающим преподавателем.'),'assignments':integer(description='Сколько эссе выдано выпуску.'),'students':integer(description='Студентов во взводах выпуска.')}, extra=True)]}
S['CohortList'] = obj({'vuc':string(),'platoons':arr(ref('Platoon')),'cohorts':arr(ref('CohortEntry'))}, ['vuc','platoons','cohorts'])
S['OwnFormAnswers'] = obj({'form':ref('Form'),'submission':obj({'id':string(),'answers':{'type':'object','additionalProperties':True},'savedSections':arr(string()),'completedAt':string(format='date-time')}, ['id','answers'])}, ['form','submission'])
S['EssayStudent'] = obj({'studentId':string(),'name':string(),'platoon':string(),'status':string(enum=['not_started','writing','submitted']),'submissionId':string(),'completedAt':string(format='date-time'),'answered':integer(),'total':integer(),'analysis':string(enum=['ready','queued','working','failed','none','off'],description='Состояние разбора сданного эссе.'),'reason':string(),'hasPrevious':boolean()}, ['studentId','name','platoon','status','answered','total'])
S['EssayCounts'] = obj({'students':integer(),'submitted':integer(),'notSubmitted':integer(),'analysed':integer(),'withoutAnalysis':integer(),'withPrevious':integer()}, ['students','submitted','notSubmitted','analysed','withoutAnalysis','withPrevious'])
S['EssayAssignment'] = obj({'id':string(),'title':string(),'created':string(format='date-time'),'active':boolean(),'cohortId':string(),'cohortName':string(),'year':integer(),'counts':ref('EssayCounts'),'students':arr(ref('EssayStudent'))}, ['id','title','created','active','cohortId','cohortName','year','counts','students'])
S['EssayOverview'] = obj({'cohorts':arr(obj({'id':string(),'name':string(),'year':integer(),'platoons':arr(string())}, ['id','name','year','platoons'])),'assignments':arr(ref('EssayAssignment')),'model':boolean(description='Подключена ли языковая модель.')}, ['cohorts','assignments','model'])
S['EssaySummary'] = obj({'assignment':obj({'id':string(),'title':string(),'cohortName':string(),'year':integer()}, extra=True),'counts':ref('EssayCounts'),'minimum':integer(description='Наименьшее число эссе с разбором для сводки.'),'canCompose':boolean(),'status':string(enum=['ready','queued','working','failed','none','off']),'reason':string(),'summary':{'type':'object','nullable':True,'description':'Текст модели (output) и то, что добавляет программа (common): по каждому общему пункту — у скольких авторов он встречается и исходные пункты разборов; notIncluded — пункты «внимание», вошедшие в никакой общий. stale — с тех пор появились другие эссе.','additionalProperties':True},'teachers':{'type':'object','description':'Блок «Преподаватели»: listed — сопоставленные со списком ВУЦ с числом благодарностей, проблем и упоминаний; unlisted — упоминания, как написал автор. Собирает программа, модель не нужна.','additionalProperties':True}}, ['assignment','counts','minimum','canCompose','status','summary','teachers'])
S['ManageStudentInput'] = obj({'action':string(enum=['move','remove','disable','enable']),'platoonId':string(description='Обязательно для move; взвод в том же ВУЦ.')}, ['action'])
S['TeacherProfileInput'] = obj({'name':string(maxLength=200),'email':string(format='email',maxLength=254),'currentPassword':string(format='password',maxLength=256,description='Обязательно при изменении электронной почты.')}, ['name','email'])
S['PasswordInput'] = obj({'currentPassword':string(format='password',maxLength=256),'newPassword':string(format='password',minLength=10,maxLength=256),'confirmPassword':string(format='password',minLength=10,maxLength=256)}, ['currentPassword','newPassword','confirmPassword'])
S['AssignmentInput'] = obj({'title':string(maxLength=200),'profileId':string(description='Шаблон теста. Не передаётся, если указан formId.'),'formId':string(description='Фиксированная анкета, например autobiography. Тогда шаблон не нужен.'),'platoon':string(maxLength=80,description='Пустая строка — всем взводам. Для эссе не передаётся.'),'cohortId':string(description='Только для эссе (formId essay): выпуск своего ВУЦ, созданный этим преподавателем; эссе получают студенты его взводов.'),'activity':boolean(default=False),'motivation':boolean(default=False),'methods':boolean(default=False,description='Методики уровней Я-концепции из таблицы 8 руководителя: добавляются целиком вместе с ВПМ (модуль V).')}, ['title'])
S['AssignmentInput']['example'] = {'title':'Входное тестирование','profileId':'ID-шаблона','platoon':'101','activity':False,'motivation':False}
S['AssignmentStatusInput'] = obj({'active':boolean()}, ['active'])
S['AssignmentStatusInput']['example'] = {'active':False}
S['AssignmentSummary'] = obj({'id':string(),'title':string(),'platoon':string(),'active':boolean(),'created':string(format='date-time'),'profileName':string()}, ['id','title','platoon','active','created','profileName'])
S['StudentAssignment'] = obj({'id':string(),'title':string(),'created':string(format='date-time'),'active':boolean(),'completed':boolean(),'status':string(),'sessionId':string(),'finishedAt':string(format='date-time'),'answered':integer(),'total':integer(),'formId':string(description='Только у анкет. answered и total тогда — заполненные разделы и их число; sessionId — id анкеты.'),'studentView':boolean(description='Только у анкет, которые автор может перечитать после сдачи (эссе): ответы — GET /students/me/submissions/{id}/answers.'),'printable':boolean(description='Только у сданной автобиографии: текст для печати — GET /students/me/submissions/{id}/autobiography.')}, ['id','title','created','active','completed','status'])
S['StudentAssignments'] = obj({'assignments':arr(ref('StudentAssignment'))}, ['assignments'])
S['Assignments'] = obj({'assignments':arr(ref('AssignmentSummary'))}, ['assignments'])
S['DashboardSession'] = obj({**{k:string() for k in ['id','studentId','assignmentId','name','platoon','title','updated']}, 'completed':boolean(),'answered':integer(),'total':integer()}, ['id','studentId','assignmentId','name','platoon','title','updated','completed','answered','total'])
S['Dashboard'] = obj({'students':arr(ref('PublicAccount')),'assignments':arr(ref('AssignmentSummary')),'sessions':arr(ref('DashboardSession'))}, ['students','assignments','sessions'])
S['Templates'] = obj({'profiles':{'type':'array','items':ref('Profile'),'nullable':True},'emptyProfile':ref('Profile')}, ['profiles','emptyProfile'])
S['SessionSummary'] = obj({**{k:string() for k in ['id','kind','participant','group','wave','observer','created','updated','profileName','profileId']},'parent':string(nullable=True),'completed':boolean(),'revoked':boolean(),'revision':integer(),'answered':integer(),'total':integer()})
S['Sessions'] = obj({'sessions':arr(ref('SessionSummary'))}, ['sessions'])
S['StartSession'] = obj({'url':string(example='/survey?id=ID-сессии')}, ['url'])
S['ConsentInput'] = obj({'value':boolean(enum=[True]),'revision':integer(minimum=0)}, ['value','revision'])
S['AnswerInput'] = obj({'value':integer(nullable=True,description='Значение в пределах шкалы вопроса. Для type=textarea: value=0 отмечает заполнение, текст передаётся в note (до 4000 символов), балл не рассчитывается. null удаляет ответ и его текст.'),'note':string(maxLength=4000),'revision':integer(minimum=0)}, ['value','revision'])
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
security={'teacher':[{'teacherCookie':[]}], 'profile':[{'teacherCookie':[]}], 'student':[{'studentCookie':[]}], 'admin':[{'basicAuth':[]}], 'auth':[]}
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
endpoint('/teachers/me/platoons','get','listPlatoons','Управление взводом','Взводы и студенты своего ВУЦ','PlatoonManagement',codes=[403])
endpoint('/admin/vucs','get','listVUCs','Администрирование','Справочник ВУЦ','VUCList',auth='admin')
endpoint('/admin/vucs','post','createVUC','Администрирование','Создать ВУЦ','VUC','CreateVUCInput','admin',status=201,codes=[409])
endpoint('/teachers/me/platoons','post','createPlatoon','Управление взводом','Создать учебный взвод','Platoon','PlatoonInput',codes=[403,409])
endpoint('/teachers/me/platoons/{id}','patch','renamePlatoon','Управление взводом','Переименовать взвод','Platoon','PlatoonInput',codes=[403,404,409])
endpoint('/teachers/me/students/{id}','patch','manageStudent','Управление взводом','Изменить взвод или доступ студента','PublicAccount','ManageStudentInput',codes=[403,404])
endpoint('/teachers/me/instructors','get','listInstructors','Управление взводом','Список преподавателей своего ВУЦ','InstructorList',description='Один список на ВУЦ: видит и правит любой преподаватель этого ВУЦ. По алфавиту. Нужен, чтобы программа сводила упоминания преподавателя в текстах студентов.',codes=[403])
endpoint('/teachers/me/instructors','post','createInstructor','Управление взводом','Добавить преподавателя в список ВУЦ','Instructor','InstructorInput',status=201,description='Полные тёзки допускаются.',codes=[403])
endpoint('/teachers/me/instructors/{id}','put','updateInstructor','Управление взводом','Исправить запись списка преподавателей','Instructor','InstructorInput',codes=[403,404])
endpoint('/teachers/me/instructors/{id}','delete','deleteInstructor','Управление взводом','Удалить запись списка преподавателей','OK',codes=[404])
endpoint('/teachers/me/cohorts','get','listCohorts','Эссе','Выпуски своего ВУЦ','CohortList',description='Выпуск — название, год и взводы. Виден всем преподавателям ВУЦ: взвод входит только в один выпуск. Изменяет, удаляет и выдаёт эссе выпуску только его создатель.',codes=[403])
endpoint('/teachers/me/cohorts','post','createCohort','Эссе','Создать выпуск','Cohort','CohortInput',status=201,codes=[403,404,409])
endpoint('/teachers/me/cohorts/{id}','put','updateCohort','Эссе','Изменить выпуск','Cohort','CohortInput',codes=[403,404,409])
endpoint('/teachers/me/cohorts/{id}','delete','deleteCohort','Эссе','Удалить выпуск','OK',description='Нельзя, если выпуску выдавали эссе.',codes=[403,404,409])
endpoint('/admin/teachers/{id}/vuc','patch','assignTeacherVUC','Администрирование','Назначить ВУЦ существующему преподавателю','PublicAccount','VUCInput','admin',description='Для преподавателей без ВУЦ. Переносит принадлежащих им студентов и взводы, отзывает старые ссылки. Повторное назначение запрещено.',codes=[404,409])
for action,title,body,status in [('register','Регистрация','RegisterInput',201),('login','Вход','LoginInput',200)]:
    op=endpoint('/auth/{role}/'+action,'post',action,'Авторизация',title,'AuthResponse',body,'auth',status,'Вход создаёт HttpOnly cookie vuc_teacher или vuc_student на 24 часа. Преподаватель регистрируется по одноразовому коду администратора. Студент — по 48-часовой ссылке преподавателя; взвод берётся из ссылки.',[401,403,404,409])
    op['responses'][str(status)]['headers']={'Set-Cookie':{'description':'HttpOnly cookie; Secure при HTTPS; SameSite=Lax.','schema':string()}}
endpoint('/auth/{role}/logout','post','logout','Авторизация','Выход',auth='auth',description='Удаляет текущий сеанс выбранной роли и очищает cookie. Тело не требуется.',codes=[404])
endpoint('/teachers/me','get','getTeacherProfile','Профиль','Профиль преподавателя','TeacherResponse',auth='profile')
endpoint('/teachers/me','patch','updateTeacherProfile','Профиль','Изменить ФИО и почту','TeacherResponse','TeacherProfileInput','profile',description='Оба поля name и email обязательны. Смена почты требует currentPassword.',codes=[409])
endpoint('/teachers/me/password','put','changeTeacherPassword','Профиль','Сменить пароль','TeacherResponse','PasswordInput','profile',description='Сохраняет вход на текущем устройстве, завершает остальные сеансы преподавателя.')
endpoint('/admin/invitations/teachers','post','inviteTeacher','Администрирование','Создать код преподавателя','TeacherInvitation','TeacherInvitationInput','admin',status=201,description='Требует Basic Auth администратора. Код действует 24 часа, привязан к почте и используется один раз.')
endpoint('/teachers/me/student-registration-links','post','createStudentRegistrationLink','Профиль','Создать ссылку для студентов взвода','StudentRegistrationLink','StudentLinkInput','profile',status=201,description='Ссылка действует 48 часов и может зарегистрировать нескольких студентов. Все они закрепляются за преподавателем и указанным взводом. Новая ссылка для того же взвода отзывает предыдущую.')
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
S['MethodsStudent']=obj({'sessionId':string(),'studentId':string(),'name':string(),'group':string()}, ['sessionId','studentId','name','group'])
S['MethodsLevelGroup']=obj({'level':string(enum=['high','medium','low',''],description='Пусто — уровень не определён: ответы на методику неполные.'),'levelText':string(),'students':arr(ref('MethodsStudent'))}, ['level','levelText','students'])
S['MethodsSummaryRow']=obj({'id':string(),'name':string(),'kind':string(enum=['method','activity']),'component':string(enum=['CS','EV','DP']),'levels':arr(ref('MethodsLevelGroup'))}, ['id','name','kind','levels'])
S['MethodsSummary']=obj({'group':string(),'groups':arr(string()),'sessions':integer(description='Учтено тестирований: последнее завершённое с методиками у каждого студента.'),'rows':arr(ref('MethodsSummaryRow'))}, ['group','groups','sessions','rows'])
methods=endpoint('/reports/methods','get','getMethodsSummary','Результаты','Уровни методик по взводу','MethodsSummary',description='По каждой шкале методик (ВПМ, Зверьков — Эйдман, эмоциональная устойчивость, волевые качества, анкета деятельностного компонента) и каждому виду деятельности — студенты на каждом уровне. Уровни считает программа по ключам методик; текста модели нет.')
methods.setdefault('parameters',[]).append({'name':'group','in':'query','required':False,'description':'Номер взвода; без него — все взводы.','schema':string()})
endpoint('/exports/answers.csv','get','exportAnswersCSV','Экспорт','Экспорт ответов в CSV')
paths['/exports/answers.csv']['get']['responses']['200']={'description':'CSV с ответами.','headers':{'Content-Disposition':{'schema':string(),'description':'attachment; filename="vuc-answers.csv"'}},'content':{'text/csv':{'schema':string()}}}
for path in ['/reports/groups','/exports/answers.csv']:
    paths[path]['get']['parameters']=[{'name':'group','in':'query','required':False,'description':'Номер взвода; без параметра — все взводы.','schema':string()}]
endpoint('/openapi.json','get','getOpenAPI','Документация','Описание OpenAPI',{'type':'object','additionalProperties':True},auth='auth')
S['Bank']['properties']['instructions']={'type':'object','additionalProperties':string()}
S['Survey']['properties']['instructions']={'type':'object','additionalProperties':string()}
S['PublicQuestion']['properties']['timeWindow']=string()
S['PublicQuestion']['properties']['responseOnly']=boolean()
S['PublicQuestion']['properties']['hint']=string()
S['PublicQuestion']['properties']['topic']=string()
S['PsychologicalAnswer']=obj({'id':string(),'text':string(),'timeWindow':string(),'selected':string(),'code':string(description='Исходный код варианта, включая NA и SKIP; пустой при отсутствии ответа.'),'note':string()})
S['PsychologicalIndicator']=obj({'id':string(),'name':string(),'limitations':string(),'total':integer(),'answered':integer(),'declined':integer(),'notApplicable':integer(),'unanswered':integer(),'answers':arr(ref('PsychologicalAnswer'))})
S['Report']['properties']['psychological']=arr(ref('PsychologicalIndicator'))
S['EvidenceChoice']=obj({'id':string(),'text':string(),'selected':string(),'interpretation':string()})
S['EvidenceTrend']=obj({'available':boolean(),'reason':string(),'previous':number(nullable=True),'delta':number(nullable=True),'previousDate':string(),'previousSessionId':string()})
S['EvidenceRow']=obj({'id':string(),'name':string(),'block':string(),'definition':string(),'limitations':string(),'selfImage':integer(nullable=True),'statements':number(nullable=True),'coverage':integer(),'total':integer(),'choices':arr(ref('EvidenceChoice')),'trend':ref('EvidenceTrend')})
S['Report']['properties']['evidence']=arr(ref('EvidenceRow'))
S['Report']['properties']['methods']={'type':'object','additionalProperties':True,'description':'Портрет по методикам (только у сессий с методиками): methods — уровни шкал по ключам, components и images — Я-реальное и Я-идеальное, signs и conditions — показатели без уровней, kinds — виды деятельности с уровнем по анкете, openAnswers — рассказы о деятельности.'}
S['Report']['properties']['interpretationNote']=string()
S['FormSubmission']['properties']['answers']={'type':'object','additionalProperties':True,'description':'Ключ — id поля верхнего уровня. choice — строка, multi — массив строк, text/textarea — строка, date/month/year — ГГГГ-ММ-ДД / ГГГГ-ММ / ГГГГ, group — массив объектов.'}
S['FormProgress']=obj({'complete':arr(string()),'missing':{'type':'object','additionalProperties':arr(string())},'total':integer()}, ['complete','missing','total'])
S['FormProgress']['description']='complete — разделы, сохранённые и без пропусков. missing — раздел: незаполненные обязательные поля; поле внутри записи называется группа.номер.поле.'
S['FormView']=obj({'submission':ref('FormSubmission'),'form':ref('Form'),'progress':ref('FormProgress')}, ['submission','form','progress'])
S['FormSectionInput']=obj({'answers':{'type':'object','additionalProperties':True,'description':'Ответы только на поля этого раздела. Скрытые развилкой поля отбрасываются, пустые значения не сохраняются.'},'revision':integer(minimum=0)}, ['answers','revision'])
S['FormCompletionInput']=obj({'revision':integer(minimum=0)}, ['revision'])
S['FormRevision']=obj({'revision':integer(),'updated':string(format='date-time')}, ['revision','updated'])
S['FormSectionResult']=obj({'revision':integer(),'updated':string(format='date-time'),'progress':ref('FormProgress')}, ['revision','updated','progress'])
S['FormCompletionResult']=obj({'revision':integer(),'updated':string(format='date-time'),'completed':boolean()}, ['revision','updated','completed'])
S['FormSuggestions']=obj({'values':arr(string())}, ['values'])
S['FormSubmissionBrief']=obj({'id':string(),'studentId':string(),'assignmentId':string(),'completed':boolean(),'completedAt':string(),'updated':string(format='date-time'),'answered':integer(description='Заполненных разделов.'),'total':integer(description='Всего разделов.')}, ['id','studentId','assignmentId','completed','updated','answered','total'])
S['FormAssignmentSummary']=obj({'id':string(),'title':string(),'platoon':string(),'active':boolean(),'created':string(format='date-time')}, ['id','title','platoon','active','created'])
S['FormOverview']=obj({'students':arr(ref('PublicAccount')),'assignments':arr(ref('FormAssignmentSummary')),'submissions':arr(ref('FormSubmissionBrief'))}, ['students','assignments','submissions'])
S['PotentialArea']=obj({'id':string(enum=['sport','creative','patriotic','science','volunteering','languages']),'title':string(),'level':string(enum=['high','medium','initial','none']),'levelText':string(),'reasons':arr(string(description='Какие ответы дали этот уровень.'))}, ['id','title','level','levelText','reasons'])
S['PotentialAward']=obj({'name':string(),'year':string()}, ['name','year'])
S['Potential']=obj({'areas':arr(ref('PotentialArea')),'high':arr(string(description='Направления с высоким уровнем.')),'awards':arr(ref('PotentialAward'))}, ['areas','high','awards'])
S['Potential']['description']='Уровни потенциала по шести направлениям, посчитанные программой по шкале из ответов студента. Общего уровня нет. Государственные награды на уровни не влияют.'
S['FormSubmissionDetail']=obj({'submission':ref('FormSubmission'),'form':ref('Form'),'student':ref('PublicAccount'),'progress':ref('FormProgress'),'potential':ref('Potential')}, ['submission','form','student','progress'])
S['PotentialStudent']=obj({'submissionId':string(),'studentId':string(),'name':string(),'platoon':string()}, ['submissionId','studentId','name','platoon'])
S['PotentialLevelGroup']=obj({'level':string(enum=['high','medium','initial','none']),'levelText':string(),'students':arr(ref('PotentialStudent'))}, ['level','levelText','students'])
S['PotentialAreaSummary']=obj({'id':string(),'title':string(),'levels':arr(ref('PotentialLevelGroup'))}, ['id','title','levels'])
S['PotentialSummary']=obj({'platoon':string(),'platoons':arr(string()),'submitted':integer(description='Учтено сданных анкет.'),'areas':arr(ref('PotentialAreaSummary'))}, ['platoon','platoons','submitted','areas'])
S['PortraitText']=obj({'output':{'type':'object','additionalProperties':True,'description':'Ответ модели по схеме задачи. У автобиографии с инструкции v4: paragraphs (два-три абзаца вывода) и consider (что учесть); до v4 — summary, potential (шесть записей area, level, facts, conclusion), hobbies (с v3), attention (use, consider). Уровни показываются из potential ответа анкеты.'},'created':string(format='date-time')}, ['output','created'])
S['PortraitView']=obj({'status':string(enum=['queued','working','failed','ready','off','none'],description='queued — в очереди, working — составляется, failed — не удалось (причина в reason), ready — готов, off — языковая модель не подключена, none — сводку никто не запрашивал (задачи по кнопке).'),'portrait':ref('PortraitText'),'reason':string()}, ['status'])
S['PortraitView']['description']='Вывод языковой модели по обезличенным сведениям. Одинаковые сведения дают один и тот же вывод.'
form_open=endpoint('/students/me/submissions/{id}','get','getFormSubmission','Анкеты','Открыть свою анкету','FormView',auth='student',description='Только собственная несданная анкета активного назначения: черновик, описание анкеты со справочниками и прогресс. Сданная анкета студенту не отдаётся (403).',codes=[403])
endpoint('/students/me/submissions/{id}/consent','put','saveFormConsent','Анкеты','Подтвердить согласие','FormRevision','ConsentInput','student',codes=[409])
endpoint('/students/me/submissions/{id}/sections/{sectionId}','put','saveFormSection','Анкеты','Сохранить раздел анкеты','FormSectionResult','FormSectionInput','student',description='Заменяет ответы только этого раздела. Черновик может быть неполным. Нужно согласие. Лимит тела 64 КиБ. Ошибка проверки содержит field — путь к полю.',codes=[403,409])
endpoint('/students/me/submissions/{id}/completion','post','completeForm','Анкеты','Сдать анкету','FormCompletionResult','FormCompletionInput','student',description='Возможна, когда сохранены все разделы и заполнено всё обязательное. Иначе 422 с перечнем разделов (sections) и пропусков (missing). После сдачи анкета закрыта.',codes=[403,409,422])
suggest=endpoint('/students/me/submissions/{id}/suggestions','get','getFormSuggestions','Анкеты','Подсказки для текстового поля','FormSuggestions',auth='student',description='До 10 значений, которые другие студенты вводили в то же поле. Только для полей с suggest.',codes=[400,403])
suggest['parameters'].extend([{'name':'field','in':'query','required':True,'description':'Идентификатор поля.','schema':string()},{'name':'q','in':'query','required':False,'description':'Часть значения без учёта регистра.','schema':string(maxLength=100)}])
endpoint('/portraits/{task}/{sourceId}','get','getPortrait','Выводы','Вывод языковой модели','PortraitView',description='Task — задача (autobiography), sourceId — id сданной анкеты. Составляется сам после сдачи; запрос только показывает состояние. Если вывода нет и он не в очереди, он ставится в очередь. Чужой или несданный источник — 404.',codes=[404])
endpoint('/portraits/{task}/{sourceId}','post','retryPortrait','Выводы','Повторить составление вывода','PortraitView',description='Снимает ошибку и ставит вывод в очередь заново. Готовый вывод возвращается как есть, модель не вызывается. Тело не требуется.',codes=[404])
endpoint('/students/me/submissions/{id}/answers','get','getOwnFormAnswers','Анкеты','Свои сданные ответы (эссе)','OwnFormAnswers',description='Только своя сданная анкета, и только такая, что показывается автору (эссе). Автобиография и черновик — 404.',auth='student',codes=[404])
endpoint('/essays','get','getEssays','Эссе','Выпуски и эссе преподавателя','EssayOverview',description='Свои выпуски и выданные им эссе; по каждому эссе — студенты взводов выпуска со статусом эссе и разбора. Разбор по человеку — GET /portraits/essay/{submissionId}.')
endpoint('/essays/assignments/{id}/summary','get','getEssaySummary','Эссе','Сводка по выдаче эссе выпуску','EssaySummary',description='Счётчики, блок «Преподаватели» и последняя составленная сводка модели. Сама не составляется.',codes=[404])
endpoint('/essays/assignments/{id}/summary','post','composeEssaySummary','Эссе','Составить сводку по выдаче','EssaySummary',description='Ставит сводку в очередь по эссе с готовым разбором (не меньше двух). Одинаковый набор разборов даёт ту же сводку без вызова модели.',codes=[400,404])
endpoint('/forms/{formId}','get','getForm','Анкеты','Описание анкеты','Form',description='Разделы, поля, развилки и справочники.',codes=[404])
endpoint('/forms/{formId}/overview','get','getFormOverview','Анкеты','Обзор анкеты: студенты, назначения, сдача','FormOverview',codes=[404])
endpoint('/submissions/{id}','get','getFormSubmissionDetail','Анкеты','Ответы студента на анкету','FormSubmissionDetail',description='Отдаётся и сданная, и начатая анкета. У сданной автобиографии есть potential — уровни потенциала.',codes=[404])
S['AutobiographyText']=obj({'student':string(description='ФИО из учётной записи.'),'paragraphs':arr(string()),'date':string(description='День сдачи по Москве, «15 сентября 2024 г.».'),'surname':string(description='Фамилия для подписи.')}, ['student','paragraphs','date','surname'])
S['AutobiographyText']['description']='Автобиография для личного дела: текст бумажного бланка от первого лица, который программа собирает по шаблону из ответов сданной автобиографии. Заново при каждом запросе, в базе не хранится; языковая модель не участвует.'
endpoint('/submissions/{id}/autobiography','get','getAutobiographyText','Анкеты','Автобиография для печати','AutobiographyText',description='Только сданная автобиография по своему назначению; несданная, чужая или другая анкета — 404.',codes=[404])
endpoint('/students/me/submissions/{id}/autobiography','get','getOwnAutobiographyText','Анкеты','Своя автобиография для печати','AutobiographyText',auth='student',description='Только своя сданная автобиография; иначе 404.',codes=[404])
potential=endpoint('/forms/{formId}/potential','get','getFormPotential','Анкеты','Потенциал по взводу','PotentialSummary',description='Только для автобиографии. По каждому направлению — студенты на каждом уровне, от высокого до «не выражен». Учитывается последняя сданная анкета каждого студента.',codes=[404])
potential['parameters'].append({'name':'platoon','in':'query','required':False,'description':'Номер взвода; без него — все взводы.','schema':string()})
spec={'openapi':'3.0.3','info':{'title':'API диагностической системы ВУЦ','version':'1.2.0','description':'Отдельные HTTP-маршруты для каждой операции. JSON-запросы требуют Content-Type: application/json; неизвестные поля отклоняются. Лимит обычных запросов 16 КиБ, шаблонов и импорта 3 МБ, плана 300 КБ.\n\nАвторизация: короткие коды преподавателей создаёт только администратор. Студенты регистрируются по 48-часовой ссылке конкретного взвода, созданной преподавателем. Вызовите /auth/teacher/login или /auth/student/login, браузер сохранит HttpOnly cookie. Basic Auth используется только административным маршрутом.\n\nОшибки возвращаются как {error, conflict?, partial?}. Изменения анкеты требуют актуальной revision.'},'servers':[{'url':'/api/v1'}],'tags':[{'name':name} for name in ['Авторизация','Администрирование','Профиль','Студент','Анкета','Анкеты','Выводы','Методика','Преподаватель','Шаблоны','Назначения','Результаты','Экспорт','Документация']],'paths':paths,'components':{'schemas':S,'securitySchemes':{'teacherCookie':{'type':'apiKey','in':'cookie','name':'vuc_teacher'},'studentCookie':{'type':'apiKey','in':'cookie','name':'vuc_student'},'basicAuth':{'type':'http','scheme':'basic','description':'Только для административных маршрутов при настроенном VUC_ADMIN_PASSWORD.'}}}}
Path('web/static/openapi.json').write_text(json.dumps(spec,ensure_ascii=False,indent=2)+'\n')
print(f'OpenAPI: {sum(len(v) for v in paths.values())} operations, {len(S)} schemas')
