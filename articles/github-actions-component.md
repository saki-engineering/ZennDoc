---
title: "GitHub ActionsにおけるStep/Job/Workflow設計論"
emoji: "🤹‍♀️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: [githubactions]
published: true
---
# この記事について
GitHub Actionsには、以下3つの実行単位が存在します。
- Workflow
- Job
- Step

パイプラインを組む中で出てくる複数個の処理を、1つの実行単位でまとめてしまうか、それとも分割するのかというのは悩むポイントかと思います。
- 一つのstepのrunフィールドにコマンドを詰め込む？それともstepを分けた方がいい？
- 一つのJobの中のstepとして記述した方がいい？それとも別のJobに定義した方がいい？
- 一つのWorkflowの中にJobをたくさん定義する？それともWorkflowを別にする？

この記事では、Workflow・Job・Stepそれぞれの性質を踏まえた上で、ベストな処理単位の選び方を考察します。

## 使用する環境・バージョン
- GitHub Actions: 2022/5/15時点での機能をもとに考察

## 読者に要求する前提知識
- GitHub ActionsでWorkflow定義に使うyamlファイルの記法がある程度わかること
- シェルスクリプトが読めること









# Stepの場合
まずはStepについて、「複数個のコマンドを1つのStepの中にまとめて書くか、それとも複数個のStepに分割するか？」を考察していきます。

## 「実行コンテキスト」という観点
GitHub Actionsでは、Stepごとに1つのシェルが与えられます。
つまり、異なるStepに書かれたコマンドは違うシェル上で実行されるということです。
```yaml
steps:
    # AAAAの定義とechoは同じシェルで行われる
    # -> echoで1111が出力される
  - name: one-step 
    run: |
      readonly AAAA=1111
      echo ${AAAA}

    # BBBBの定義とechoは別シェルで実行される
    # -> echoで2222は出力されない
  - name: separate-step-1
    run: readonly BBBB=2222
  - name: separate-step-2
    run: echo ${BBBB}
```
そのため、基本的には**異なるシェルで実行してしまうとうまくいかないものはまとめる**という方針をとることになります。
では、異なるシェルにしてしまうとうまくいかない・1つのStepに処理をまとめざるを得ないというのはどのようなパターンが考えられるでしょうか。

### Stepでの実行結果を外部にシェアするとき
あるStepでの実行結果を他のStepが参照するためには、[環境変数への書き込み](https://docs.github.com/ja/actions/using-workflows/workflow-commands-for-github-actions#setting-an-environment-variable)、もしくは[`set-output`コマンド](https://docs.github.com/ja/actions/using-workflows/workflow-commands-for-github-actions#setting-an-output-parameter)の実行といった準備が必要です。
この場合、「Step本来の処理」「結果を出力するためのコマンド」は同じStepの中に書く必要があります。
```yaml
# 環境変数への書き込み
steps:
  - name: write-env
    run: |
      CURRENT_DIR=$(pwd)
      echo "CURRENT_DIR=${CURRENT_DIR}" >> $GITHUB_ENV

  - name: read-write-env-result
    run: echo ${{ env.CURRENT_DIR }}
```
```yaml
# set-outputコマンドの実行
steps:
  - name: set-step-output
    id: stepid
    run: |
      CURRENT_DIR=$(pwd)
      echo '::set-output name=CURRENT_DIR::${CURRENT_DIR}'

  - name: read-step-output
    run: echo ${{ steps.stepid.outputs.CURRENT_DIR }}
```

### コマンド実行に失敗したとしてもWorkflowをFAILさせたくない場合
「定義されているStepの中の少なくとも1つが、終了ステータス0以外になった」場合には、WorkflowはFAIL扱いになります。
[ステータスチェック関数](https://docs.github.com/ja/actions/learn-github-actions/expressions#status-check-functions)というものを使って「Workflowが失敗していたとしてもStepやJobを実行させる」ということはできますが、それでも「とあるStep/Jobが異常なステータスで終了した」という事実は消すことができません。
つまり、プログラムでいう`panic`-`recover`のようなこと、「とあるStepでの異常終了を他のStepで握り潰す」ということはGitHub Actionsではできないのです。
```yaml
steps:
  - name: step1
    run: command1 # もしここが異常ステータスで終了したら

  - name: step2
    run: command2 # このステップは実行されないが

  - name: step3
    if: failure() # ステータスチェック関数failure()によるif文条件をつけることで
    run: command3 # このステップは実行される

# Workflow全体としてはFAIL扱い
# step3でstep1の異常ステータスを握り潰して、WorkflowをSUCCESSにさせるような処理は作れない
```
:::details [余談]Go言語におけるpanic-recover
```go
func main() {
	defer func() {
		err := recover() // これがあればpanicが起きてもexit 0で終わらせられる
		fmt.Println(err)
	}()
	panic("panic occur")
}
```
:::

そのため、WorkflowをFAILさせないために「失敗したコマンドのエラーをcatchする」という処理は同じStepの中に組み込む必要があります。
```yaml
steps:
  - name: may-fail
    run: may-fail-command || if-failed-run-command
```

## 「GUIでの見やすさ」という観点
かといって、「別のシェルで実行しても問題ないものは、全て別のStepに分割しよう！」というのも考えものです。
なぜならば 「GitHub Actionsの処理結果を確認するGUIでは、各StepごとにSUCCESS/FAILが表示される」ようになっているからです。

例えば、以下のような処理を考えます。
1. とある差分を`git add`する
2. 1の内容をコミットする
3. 2のコミットをpushする

この3つの処理は「差分をpushするための操作」とひとまとめにして捉えるのが自然です。
このような処理まで一つ一つ別のStepに分割してしまうと、「Jobの中にたくさんStepがあって見るのが大変だなあ……」ということになります。
適度にかつ常識の範囲で、まとめられるものはまとめていくのが運用する上では良いかと思います。






# Jobの場合
続いてはJobについて、「複数個のStepを、1つのJobの中にまとめて定義するか、それとも複数個のJobに分割するか？」を考察していきます。

## 「実行コンテキスト」という観点
GitHub Actionsでは、1つのJobごとに1つのRunner(=コンテナ実行環境)が用意されます。
```yaml
jobs:
  # Job1用のコンテナ環境が用意される
  Job1:
    runs-on: ubuntu-latest
    steps:
      - name: step1
        run: echo a

  # Job2用のコンテナ環境が用意される
  Job2:
    runs-on: ubuntu-latest
    steps:
      - name: step1
        run: echo b
```
そのため、**同じマシンの環境を引き継がないといけない処理**については同一Jobの中にまとめるしかありません。

(例)コンテナイメージをbuild -> buildしたものをpush

## 「re-runの単位」という観点
StepとJobはどちらも「コマンド処理をまとめたもの」ですが、WorkflowがFAILしたときにどこからre-runすることができるか？という点については違いがあります。

### Stepのre-run
1つのJobの中に「成功したStep」と「失敗したStep・左の失敗に伴ってSkipされたStep」があった場合に、**後者のみをre-runさせる方法は存在しません。**
```yaml
steps:
  - name: step1 # 成功
    run: command1

  - name: step2 # 失敗
    run: command2

  - name: step3 # step2が失敗したのでskipされた
    run: command3

# -> step2,3だけをre-runさせる方法はない 
```
GitHub Actionsに用意されている[re-runの機能](https://docs.github.com/ja/actions/managing-workflow-runs/re-running-workflows-and-jobs)は、
- Workflow全体をre-run
- FAILしたJobのみをre-run

の2つであり、「FAILしたJobを、FAILしたStepから再開」ということはできないのです。

### Jobのre-run
前述の通り、1つのWorkflowの中に「成功したJob」と「失敗したJob・左の失敗に伴ってSkipされたJob」があった場合に、**後者のみをre-runさせることができます。**

### re-runの単位を考慮した設計思想
以上のことから、「**GitHub ActionsでいうJobは、Atomicな処理単位である**」ということができます。

これを踏まえると、例えば以下のような判断ができるようになります。
- コンテナイメージのPushと、それを用いたDeploy
  - Pushは成功してDeployが失敗したとしても、やり直したいのはDeployのみなので、PushのJobとDeployのJobに分割










# Workflowの場合
最後にWorkflowについて、「複数個のJobを、1つのWorkflowの中にまとめて書くか、それとも分割するか？」を考察していきます。
「分けたいな」と思うパターンは分けて正解のことがほとんどなので、ここからは「1つのWorkflowにまとめようと思っているけど、それは本当にベストなのか？」という観点で論じていきます。

## 「実行順序」という観点
Workflowの重要な性質は、「**ある同一のトリガ条件で実行される複数個のWorkflowの実行順序をこちらから決める手段は存在しない**」ということです。
そのため「本当は分けられたら分けたいんだけど、実行順を制御するために1つのWorkflowにせざるを得ないのか？」という場面になることがあります。

### 実行順が決定できなくて困るパターン
具体例を出して説明すると、以下のようなテスト用ワークフローとデプロイ用Workflowが存在したとします。
テストとデプロイで目的が全然別ですので、素直に考えるならばWorkflowも分けたいところです。
```yaml
name: Test Workflow
on: 
  push:

jobs:
  Test:

# (以下略)
```
```yaml
name: Deploy Workflow
on: 
  push:
    branches:
      - "deployment/**"

jobs:
  Deploy:

# (以下略)
```
しかしこの場合、`deployment/**`のブランチにpushがあった場合には、両方のWorkflowが発火することになります。
そしてそのようなときに「テスト用のWorkflowが終わってから、デプロイ用のWorkflowを実行させたい」という制御をすることは不可能です。

このケースでは、一見Workflowをテスト用とデプロイ用の2つに分割するのは難しそうにみえます。
しかし、これにはいい解決策があるのです。

### 解決策: Reusable Workflowの利用
`on`条件に`workflow_call`を指定することで、[他のWorkflowからそのWorkflowをJobの形で呼び出す](https://docs.github.com/ja/actions/using-workflows/reusing-workflows)ということができるようになります。

以下に「デプロイ用のWorkflowから、テスト用のWorkflowを呼び出す」ための記述を示します。
```diff:test_workflow.yaml
name: Test Workflow
on: 
  push:
+   branches-ignore:
+     - "deployment/**"
+  workflow_call:

# ->「deployment/**以外のブランチにpushがあった場合」or「他Workflowから呼び出された場合」に発火する

jobs:
  Test:

# (以下略)
```
```diff:yaml
name: Deploy Workflow
on: 
  push:
    branches:
      - "deployment/**"
jobs:
+ Test:
+   uses: ./.github/workflows/test_workflow.yaml
  Deploy:
+   needs: [Test]

# (以下略)
```
ここでのポイントは「`branches-ignore`条件を使って、テスト用Workflowとデプロイ用Workflowの発火条件を互いに排反にしている」というところです。
こうすることで、
- テスト用Workflowの発火条件のとき: テスト用Workflowのみ実行
- デプロイ用Workflowの発火条件のとき: テスト用Workflowの全Job -> デプロイJobの順で実行

という条件処理を実現することができます。

このようにReusable Workflowを利用することで、「テスト用とデプロイ用で分ける」という直感的なWorkflow設計をすることができました。

## 「Workflowの用途」という観点
ある用途のために作られるJobは一つのWorkflowにまとめたくなりますし、実際その判断は正しいことがほとんどです。
ただし、その「用途」の区切りも適切に設定しないと、Workflowのロジックが肥大し複雑になってしまうことがあります。

### デプロイWorkflowを環境ごとに用意するか否か
ここでは一例として、デプロイパイプラインを取り上げます。
気軽にデプロイを行いたいDev環境と、動作確認がきちんとできたものを確実に出したいPrd環境では、同じデプロイといえど違う性質を持つことがあります。
- Prdリリース時だけ、該当コミットにタグづけを行いたい
- Prdリリース時は、ソースコードから成果物のビルドを行うのではなく、既にビルド済みの成果物を流用したい

DevとPrdであまりにもフローが異なる場合、どうしても「一方の環境のときにだけ使用するStep/Job」というのが出てくることになります。
しかしGitHub Actionsというのは**そもそも分岐処理を表現するのに不向きな性質を持っています。**
以下、その件について説明します。

### GitHub Actionsの分岐処理表現力
Workflowの中に出てくる処理単位「Step」と「Job」は、どちらも`if`を用いて「ある条件のときに処理を実行するかしないか」を制御することができます。
しかしStep/Job双方の性質を踏まえると、`if`を用いたとしても柔軟な分岐処理を実装するのは難しいと言わざるを得ません。

#### Stepでできる分岐表現
Stepは「記述された順番通りに直接に実行される処理を記述する」場所なので、**forkを表現するのには不向き**です。
`if`を用いて地道に条件を記述していくことでできなくはないですが、可読性も損なわれますしあまりいいコードとは呼べないでしょう。
```yaml
# ifを使って地道にfork/joinしている例
steps:
  - name: chech-env
    id: chech-env
    run: |
      ENV=$(sh ./check-env.sh)
      echo '::set-output name=ENV::${ENV}'

  # dev環境用にfork
  - name: dev-step-1
    if: steps.chech-env.outputs.ENV == 'dev'

  - name: dev-step-2
    if: steps.chech-env.outputs.ENV == 'dev' # step1と同じif文が必要

  # prd環境用にfork
  - name: prd-step-1
    if: steps.chech-env.outputs.ENV == 'prd'

  - name: prd-step-2
    if: steps.chech-env.outputs.ENV == 'prd' # step1と同じif文が必要

  # dev, prd環境個別の処理が終わってjoin
  # -> これに特別な記述は必要ない
  - name: join-step-1

  - name: join-step-2
```

ただ逆に、joinに関しては容易に行うことができます。
同じJobに属するStepは、`if`文によるスキップさえ行われなければ全て実行されることになるので、上の例でいうと「Dev用の処理またはPrd用の処理が実行されたら……」という条件文をわざわざ`join-step-xx`のStepに記述する必要がないのです。

#### Jobでできる分岐表現
Jobを使う場合には、forkに関しては`needs`と`if`を用いて簡潔に表現することができます。
```yaml
jobs:
  CheckEnvJob:
    steps:
      - name: chech-env
        id: chech-env
        run: |
          ENV=$(sh ./check-env.sh)
          echo '::set-output name=ENV::${ENV}'
    outputs:
      env: {{ steps.check-env.outputs.ENV }}

  # dev環境用にfork
  DevJob:
    needs: CheckEnvJob
    if: needs.CheckEnvJob.outputs.env == 'dev' # if文を書くのはここ1回だけ
    steps:
      - name: dev-step-1
      - name: dev-step-2

  # prd環境用にfork
  PrdJob:
    needs: CheckEnvJob
    if: needs.CheckEnvJob.outputs.env == 'prd' # if文を書くのはここ1回だけ
    steps:
      - name: prd-step-1
      - name: prd-step-2
  
  # dev, prd環境個別の処理が終わってjoin
  JoinedJob:
    needs: [DevJob, PrdJob]
    if: always() && contains(needs.*.result, 'success') # if文を書くのはここ1回だけ
    steps:
      - name: join-step-1
      - name: join-step-2

    # (補足)
    # if: contains(needs.*.result, 'success')は以下と同値です
    # if: (needs.DevJob.result == 'success' || needs.PrdJob.result == 'success')
```

その代わり、**Jobはjoinが苦手です**。
forkした複数個の処理を待つためには`needs`が必要なのですが、本来`needs`は「指定したJob**全てがsuccessになったら**」という意味なので、「forkした処理の**どれか一つがsuccessになったら**」というOR条件を表現するためには、ステータスチェック関数の`always`を使って無理やり実現することになります。
コードとしてもあまり直感的ではないですし、あまりやりたくない方法です。
```yaml
# これだと、DevJob, PrdJobどちらからSkipされたときにJoinedJobが実行されない
JoinedJob:
  needs: [DevJob, PrdJob]
  if: contains(needs.*.result, 'success')

# alwaysをつけることで、needsで指定したJobの結果によらずcontainsの判定が行われる
# -> それに基づきJoinedJobの実行有無が決定
JoinedJob:
  needs: [DevJob, PrdJob]
  if: always() && contains(needs.*.result, 'success')
```

### 大掛かりな処理分岐をするぐらいなら、Workflow自体を分けるのが得策
このように、StepとJobは双方ともに条件分岐を実装する上での弱点が存在します。
- Step: joinはできるがforkが苦手
- Job: forkはできるがjoinが苦手

そのため、条件によってStepやJobの挙動を変えたいというのであれば、まずまっさきに考えるべきは「コマンド引数を用いて、どの条件でも使えるような処理に書き直せないか？」とワークフロー上での分岐を避ける方向であって、条件分岐は最後の手段です。
もし条件分岐に頼らざるを得ないという結論になったとしても、それでコードの可読性を損なうような多量の`if`が必要になるならば、素直に条件ごとにWorkflow自体を分割するのが得策なのではないでしょうか。

幸いにも「異なるWorkflow上に同様のJobを実行したい」というニーズのためにReusable Workflowが存在するので、「Workflowを分けると同じコードを複数箇所に書くことになるのでは」という部分については心配する必要はありません。







# まとめ
ここまでの話をまとめると、「Step/Job/Workflowを分けるべきか？」というのは以下のような基準になります。
- Step
  - 違うシェルで実行するとおかしくなるものについては1つのStepにまとめる一択
  - ↑のような事情がないならば、見やすさ読みやすさを考えて区切っていけばOK
- Job
  - 違うマシン上で動かすことができないStepは1つのJob上にまとめる一択
  - ↑のような事情がないならば、Workflow自体がFAILしたときにどこからre-runさせるのが良いか？というのをもとにJobを分割する
- Workflow
  - **分けたいなと思ったときは分けるのが正解**
  - 同一トリガで発火するWorkflowの間に実行順序を付けたいという場合にも、Reusable Workflowを利用してリファクタすれば別Workflowのままにすることができるので、無理やりWorkflowを結合しなくてもOK
  - StepとJobに複雑な条件分岐を入れるくらいなら、そもそもWorkflow自体を分けてしまうのが得策


また、今回の考察を進めるにあたって明らかになったGitHub Actionsの各種性質についてもまとめます。
- 異なるStepで`panic`-`recover`することはできない
- re-runの単位がJobからなので、Atomicな処理単位はJobと捉えるべき
- ある同一のトリガ条件で実行される複数個のWorkflowの実行順序をこちらから決める手段は存在しない
- **GitHub ActionsのWorkflowは、条件分岐を記述するのに向いていない**

条件分岐の記述が思うようにできないことについては、通常のプログラミング言語を扱う感覚で向かうと不便と感じるかもしれません。
ですがこの性質によって「Workflowを細かく分けさせて、その代わり1つ1つをシンプルに保つ」ということを自然と行うことができると考えると、悪くない仕様なのかなと個人的には思います。
