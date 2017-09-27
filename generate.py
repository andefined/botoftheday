from datetime import datetime, timedelta
import argparse
import numpy as np
import pandas as pd

import seaborn as sns
import matplotlib.pyplot as plt

now = datetime.now()
d = now - timedelta(days=1)

timeTitle = d.strftime('%Y-%m-%d %I %p') + ' — ' + now.strftime('%Y-%m-%d %I %p')

parser = argparse.ArgumentParser()
parser.add_argument('-u', dest='user', default='adonisgeorgiadi')
results = parser.parse_args()

activity = 'data/' + results.user + '-activity-'
asdata = pd.read_csv(activity + 'source-' + now.strftime('%Y-%m-%d') + '.csv')
atdata = pd.read_csv(activity + 'target-' + now.strftime('%Y-%m-%d') + '.csv')

mentions = 'data/' + results.user + '-mentions-'
msdata = pd.read_csv(mentions + 'source-' + now.strftime('%Y-%m-%d') + '.csv')
mtdata = pd.read_csv(mentions + 'target-' + now.strftime('%Y-%m-%d') + '.csv')

msdata = msdata.query('count > 1')
msdata = msdata.sort_values(by='count', ascending=False)
mtdata = mtdata.query('count > 1')
mtdata = mtdata.sort_values(by='count', ascending=False)
mdata = pd.concat([msdata, mtdata])
mdata = mdata.sort_values(by='count', ascending=False)
source = asdata['bot'][0]
target = atdata['bot'][0]

def createActivityPlot(data):
    sns.set_style('whitegrid', {
        # {'font.family': ['Roboto']}
        'xtick.color': '0.15',
        'ytick.color': '0.15',
        'ytick.direction': 'in'
    })
    sns.set_context('paper')
    f, ax = plt.subplots(figsize=(9,6))
    title = source + ' / ' + target + ' Activity ' + timeTitle
    sns.barplot(
        hue='bot',
        x='created_at',
        y='count',
        data=data,
        palette=['#646464', '#ff0000'],
        dodge=True
    )

    ax.legend(ncol=1, loc='upper left', frameon=False)
    ax.set_xlabel('Hour of the Day')
    ax.set_ylabel('Tweets / Activity')
    sns.despine(left=True, bottom=True)
    plt.suptitle(title)
    plt.tight_layout(pad=4)
    plt.savefig(activity + now.strftime('%Y-%m-%d') + '.png')

def createMentionsPlot(data):
    sns.set_style('whitegrid', {
        'xtick.color': '0.15',
        'ytick.color': '0.15',
        'ytick.direction': 'in'
    })
    sns.set_context('paper')
    f, ax = plt.subplots(figsize=(9,6))
    title = source + ' / ' + target + ' Actions (RTs, RPs, QTs) ' + timeTitle
    sns.barplot(
        hue='bot',
        x='count',
        y='user',
        data=data[:40],
        palette=['#646464', '#ff0000'],
        dodge=True

    )
    ax.legend(ncol=1, loc='lower right', frameon=False)
    ax.set_xlabel('Mentions (RTs, RPs, QTs)')
    ax.set_ylabel('@ Top Users')
    sns.despine(left=True, bottom=True)
    plt.suptitle(title)
    plt.tight_layout(pad=4)
    plt.savefig(mentions + now.strftime('%Y-%m-%d') + '.png')

createActivityPlot(pd.concat([asdata,atdata]))
createMentionsPlot(mdata)
