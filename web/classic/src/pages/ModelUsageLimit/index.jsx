import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Banner, Button, Empty, Popconfirm, Select, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';
import CardPro from '../../components/common/ui/CardPro';
import CardTable from '../../components/common/ui/CardTable';
import AddLimitModal from './modals/AddLimitModal';
import EditLimitModal from './modals/EditLimitModal';
import { createCardProPagination } from '../../helpers/utils';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const { Text } = Typography;

const LimitDescription = ({ total, t }) => (
  <div className='flex flex-col gap-1'>
    <Text strong style={{ fontSize: 16 }}>
      {t('Model Usage Limits')}
    </Text>
    <Text type='tertiary'>
      {t('Configure daily / weekly / monthly token and request limits by user and model. Exceeded requests return 429 directly.')}
    </Text>
    <Text type='secondary'>{t('Current Config Count')}: {total || 0}</Text>
  </div>
);

const HitsDescription = ({ total, t }) => (
  <div className='flex flex-col gap-1'>
    <Text strong style={{ fontSize: 16 }}>
      {t('Hit Records')}
    </Text>
    <Text type='tertiary'>
      {t('View which user, model, limit type, and period triggered the over-limit event.')}
    </Text>
    <Text type='secondary'>{t('Current Record Count')}: {total || 0}</Text>
  </div>
);

const getLimitTypeColor = (value) => {
  if (value === 'token') return 'blue';
  if (value === 'request') return 'purple';
  return 'grey';
};

const getPeriodTypeColor = (value) => {
  if (value === 'daily') return 'green';
  if (value === 'weekly') return 'orange';
  if (value === 'monthly') return 'red';
  return 'grey';
};

const formatUsage = (used, limit) => {
  if (limit > 0) {
    return `${used || 0}/${limit}`;
  }
  return '-';
};

const ModelUsageLimit = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();

  const [activeTab, setActiveTab] = useState('limits');
  const [limits, setLimits] = useState([]);
  const [limitLoading, setLimitLoading] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [limitPage, setLimitPage] = useState(1);
  const [limitSize, setLimitSize] = useState(10);
  const [limitTotal, setLimitTotal] = useState(0);
  const [hits, setHits] = useState([]);
  const [hitLoading, setHitLoading] = useState(false);
  const [hitKeyword, setHitKeyword] = useState('');
  const [hitPeriodType, setHitPeriodType] = useState('');
  const [hitLimitType, setHitLimitType] = useState('');
  const [hitPage, setHitPage] = useState(1);
  const [hitSize, setHitSize] = useState(10);
  const [hitTotal, setHitTotal] = useState(0);
  const [showAdd, setShowAdd] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [editingRow, setEditingRow] = useState(null);

  const loadLimits = async (page = limitPage, size = limitSize) => {
    setLimitLoading(true);
    try {
      const res = await API.get('/api/user_model_usage_limit', {
        params: { keyword, p: page, page_size: size },
      });
      const { success, data, message } = res.data;
      if (success) {
        setLimits(data.items || []);
        setLimitTotal(data.total || 0);
        setLimitPage(page);
        setLimitSize(size);
      } else {
        showError(message);
      }
    } finally {
      setLimitLoading(false);
    }
  };

  const loadHits = async (page = hitPage, size = hitSize) => {
    setHitLoading(true);
    try {
      const res = await API.get('/api/user_model_usage_limit/hits', {
        params: {
          keyword: hitKeyword,
          period_type: hitPeriodType,
          limit_type: hitLimitType,
          p: page,
          page_size: size,
        },
      });
      const { success, data, message } = res.data;
      if (success) {
        setHits(data.items || []);
        setHitTotal(data.total || 0);
        setHitPage(page);
        setHitSize(size);
      } else {
        showError(message);
      }
    } finally {
      setHitLoading(false);
    }
  };

  useEffect(() => {
    loadLimits(1, limitSize);
    loadHits(1, hitSize);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleDelete = async (row) => {
    const res = await API.delete(`/api/user_model_usage_limit/${row.id}`);
    const { success, message } = res.data;
    if (success) {
      showSuccess(message || t('Deleted successfully'));
      const shouldFallbackPage = limits.length === 1 && limitPage > 1;
      loadLimits(shouldFallbackPage ? limitPage - 1 : limitPage, limitSize);
    } else {
      showError(message);
    }
  };

  const limitColumns = [
    {
      title: t('User'),
      key: 'user',
      render: (_, row) => (
        <div className='flex flex-col'>
          <Text strong>{row.username || '-'}</Text>
          <Text type='tertiary'>ID: {row.user_id}</Text>
        </div>
      ),
    },
    {
      title: t('Model'),
      dataIndex: 'model_name',
      render: (value) => <Text>{value || '-'}</Text>,
    },
    {
      title: t('Daily Limits'),
      key: 'daily',
      render: (_, row) => (
        <div className='flex flex-col gap-1'>
          <Text size='small'>Tokens: {formatUsage(row.daily_token_used, row.daily_token_limit)}</Text>
          <Text size='small'>{t('Request Count')}: {formatUsage(row.daily_request_used, row.daily_request_limit)}</Text>
        </div>
      ),
    },
    {
      title: t('Weekly Limits'),
      key: 'weekly',
      render: (_, row) => (
        <div className='flex flex-col gap-1'>
          <Text size='small'>Tokens: {formatUsage(row.weekly_token_used, row.weekly_token_limit)}</Text>
          <Text size='small'>{t('Request Count')}: {formatUsage(row.weekly_request_used, row.weekly_request_limit)}</Text>
        </div>
      ),
    },
    {
      title: t('Monthly Limits'),
      key: 'monthly',
      render: (_, row) => (
        <div className='flex flex-col gap-1'>
          <Text size='small'>Tokens: {formatUsage(row.monthly_token_used, row.monthly_token_limit)}</Text>
          <Text size='small'>{t('Request Count')}: {formatUsage(row.monthly_request_used, row.monthly_request_limit)}</Text>
        </div>
      ),
    },
    {
      title: t('Actions'),
      key: 'action',
      render: (_, row) => (
        <Space>
          <Button
            icon={<IconEdit />}
            size='small'
            theme='borderless'
            type='tertiary'
            onClick={() => {
              setEditingRow(row);
              setShowEdit(true);
            }}
          >
            {t('Edit')}
          </Button>
          <Popconfirm
            title={t('Confirm deleting this limit configuration?')}
            content={`${row.username || row.user_id} / ${row.model_name}`}
            onConfirm={() => handleDelete(row)}
          >
            <Button icon={<IconDelete />} size='small' theme='borderless' type='danger'>
              {t('Delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const hitColumns = [
    {
      title: t('Time'),
      dataIndex: 'created_at',
      render: (value) => timestamp2string(Math.floor(value / 1000)),
    },
    {
      title: t('User'),
      key: 'user',
      render: (_, row) => (
        <div className='flex flex-col'>
          <Text strong>{row.username || '-'}</Text>
          <Text type='tertiary'>ID: {row.user_id}</Text>
        </div>
      ),
    },
    { title: t('Model'), dataIndex: 'model_name' },
    {
      title: t('Limit Type'),
      dataIndex: 'limit_type',
      render: (value) => <Tag color={getLimitTypeColor(value)}>{value || '-'}</Tag>,
    },
    {
      title: t('Period'),
      dataIndex: 'period_type',
      render: (value) => <Tag color={getPeriodTypeColor(value)}>{value || '-'}</Tag>,
    },
    {
      title: t('Usage'),
      key: 'usage',
      render: (_, row) => (
        <div className='flex flex-col gap-1'>
          <Text size='small'>{t('Used')}: {row.used_value}</Text>
          <Text size='small'>{t('Limit')}: {row.limit_value}</Text>
        </div>
      ),
    },
    {
      title: 'Request ID',
      dataIndex: 'request_id',
      render: (value) => (
        <Text type='tertiary' style={{ maxWidth: 220, display: 'inline-block' }} ellipsis={{ showTooltip: true }}>
          {value || '-'}
        </Text>
      ),
    },
  ];

  const limitsSearchArea = (
    <div className='flex flex-col md:flex-row gap-2 w-full md:items-center md:justify-between'>
      <div className='flex flex-col md:flex-row gap-2 w-full md:max-w-[520px]'>
        <input
          className='semi-input'
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          placeholder={t('Search user ID, username, or model name')}
        />
        <Button onClick={() => loadLimits(1, limitSize)}>{t('Search')}</Button>
      </div>
      <Button icon={<IconPlus />} type='primary' theme='solid' onClick={() => setShowAdd(true)}>
        {t('Create Limit')}
      </Button>
    </div>
  );

  const hitsSearchArea = (
    <div className='flex flex-col lg:flex-row gap-2 w-full lg:items-center lg:justify-between'>
      <div className='flex flex-col md:flex-row gap-2 w-full'>
        <input
          className='semi-input'
          value={hitKeyword}
          onChange={(e) => setHitKeyword(e.target.value)}
          placeholder={t('Search user ID, username, or model name')}
        />
        <Select value={hitPeriodType} onChange={setHitPeriodType} placeholder={t('All Periods')} style={{ width: 140 }}>
          <Select.Option value=''>{t('All Periods')}</Select.Option>
          <Select.Option value='daily'>daily</Select.Option>
          <Select.Option value='weekly'>weekly</Select.Option>
          <Select.Option value='monthly'>monthly</Select.Option>
        </Select>
        <Select value={hitLimitType} onChange={setHitLimitType} placeholder={t('All Types')} style={{ width: 140 }}>
          <Select.Option value=''>{t('All Types')}</Select.Option>
          <Select.Option value='token'>token</Select.Option>
          <Select.Option value='request'>request</Select.Option>
        </Select>
        <Button onClick={() => loadHits(1, hitSize)}>{t('Search')}</Button>
      </div>
    </div>
  );

  return (
    <div className='mt-[60px] px-2'>
      <div className='flex flex-col gap-4'>
        <Banner fullMode={false} type='info' description={t('Provide usable configuration first, then observe whether finer-grained filters or export are needed.')} closeIcon={null} />

        <CardPro
          type='type3'
          descriptionArea={activeTab === 'limits' ? <LimitDescription total={limitTotal} t={t} /> : <HitsDescription total={hitTotal} t={t} />}
          tabsArea={
            <Space>
              <Button theme={activeTab === 'limits' ? 'solid' : 'light'} type={activeTab === 'limits' ? 'primary' : 'tertiary'} onClick={() => setActiveTab('limits')}>
                {t('Limits')}
              </Button>
              <Button theme={activeTab === 'hits' ? 'solid' : 'light'} type={activeTab === 'hits' ? 'primary' : 'tertiary'} onClick={() => setActiveTab('hits')}>
                {t('Hit Records')}
              </Button>
            </Space>
          }
          searchArea={activeTab === 'limits' ? limitsSearchArea : hitsSearchArea}
          paginationArea={createCardProPagination({
            currentPage: activeTab === 'limits' ? limitPage : hitPage,
            pageSize: activeTab === 'limits' ? limitSize : hitSize,
            total: activeTab === 'limits' ? limitTotal : hitTotal,
            onPageChange: (page) => (activeTab === 'limits' ? loadLimits(page, limitSize) : loadHits(page, hitSize)),
            onPageSizeChange: (size) => (activeTab === 'limits' ? loadLimits(1, size) : loadHits(1, size)),
            isMobile,
            t,
          })}
        >
          {activeTab === 'limits' ? (
            <CardTable
              rowKey='id'
              columns={limitColumns}
              dataSource={limits}
              loading={limitLoading}
              pagination={false}
              hidePagination
              empty={<Empty description={t('No limit configurations yet')} />}
            />
          ) : (
            <CardTable
              rowKey='id'
              columns={hitColumns}
              dataSource={hits}
              loading={hitLoading}
              pagination={false}
              hidePagination
              empty={<Empty description={t('No hit records yet')} />}
            />
          )}
        </CardPro>
      </div>

      <AddLimitModal visible={showAdd} handleClose={() => setShowAdd(false)} refresh={() => loadLimits(limitPage, limitSize)} />

      <EditLimitModal
        visible={showEdit}
        handleClose={() => {
          setShowEdit(false);
          setEditingRow(null);
        }}
        editingRow={editingRow}
        refresh={() => loadLimits(limitPage, limitSize)}
      />
    </div>
  );
};

export default ModelUsageLimit;
