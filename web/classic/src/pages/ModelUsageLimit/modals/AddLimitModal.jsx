import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Col, Form, Row, SideSheet, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconClose, IconSave } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Title } = Typography;

const AddLimitModal = ({ visible, handleClose, refresh }) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [userOptions, setUserOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const isMobile = useIsMobile();

  const getInitValues = () => ({
    user_id: '',
    model_name: '',
    daily_token_limit: 0,
    daily_request_limit: 0,
    weekly_token_limit: 0,
    weekly_request_limit: 0,
    monthly_token_limit: 0,
    monthly_request_limit: 0,
  });

  useEffect(() => {
    if (!visible) return;
    const loadOptions = async () => {
      const [usersRes, modelsRes] = await Promise.all([
        API.get('/api/user/?p=1&page_size=1000'),
        API.get('/api/user/models'),
      ]);
      if (usersRes.data.success) {
        const items = usersRes.data.data?.items || [];
        setUserOptions(
          items.map((item) => ({
            label: `${item.username || item.display_name || item.id} (${item.id})`,
            value: String(item.id),
          })),
        );
      }
      if (modelsRes.data.success) {
        const items = modelsRes.data.data || [];
        setModelOptions(items.map((item) => ({ label: item, value: item })));
      }
    };
    loadOptions();
  }, [visible]);

  const submit = async (values) => {
    const payload = {
      ...values,
      user_id: Number(values.user_id),
    };
    setLoading(true);
    const res = await API.post('/api/user_model_usage_limit', payload);
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('Limit configuration created successfully!'));
      formApiRef.current?.setValues(getInitValues());
      refresh();
      handleClose();
    } else {
      showError(message);
    }
    setLoading(false);
  };

  return (
    <SideSheet
      placement='left'
      title={
        <Space>
          <Tag color='green' shape='circle'>{t('Create')}</Tag>
          <Title heading={4} className='m-0'>{t('Create Limit')}</Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={isMobile ? '100%' : 600}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('Submit')}
            </Button>
            <Button theme='light' type='primary' onClick={handleClose} icon={<IconClose />}>
              {t('Cancel')}
            </Button>
          </Space>
        </div>
      }
      onCancel={handleClose}
    >
      <div className='p-4'>
        <Form
          initValues={getInitValues()}
          getFormApi={(api) => {
            formApiRef.current = api;
          }}
          onSubmit={submit}
          labelPosition='left'
          labelAlign='right'
          labelWidth={130}
        >
          <Row gutter={12}>
            <Col span={12}>
              <Form.Select
                field='user_id'
                label={t('User ID')}
                placeholder={t('Select existing user')}
                optionList={userOptions}
                rules={[{ required: true, message: t('Required') }]}
              />
            </Col>
            <Col span={12}>
              <Form.Select
                field='model_name'
                label={t('Model Name')}
                placeholder={t('Select existing model')}
                optionList={modelOptions}
                rules={[{ required: true, message: t('Required') }]}
              />
            </Col>
          </Row>

          <div className='mt-4'>
            <Tag color='blue'>{t('Daily Limits')}</Tag>
          </div>
          <Row gutter={12} className='mt-2'>
            <Col span={12}>
              <Form.InputNumber field='daily_token_limit' label='Tokens' min={0} />
            </Col>
            <Col span={12}>
              <Form.InputNumber field='daily_request_limit' label={t('Request Count')} min={0} />
            </Col>
          </Row>

          <div className='mt-4'>
            <Tag color='orange'>{t('Weekly Limits')}</Tag>
          </div>
          <Row gutter={12} className='mt-2'>
            <Col span={12}>
              <Form.InputNumber field='weekly_token_limit' label='Tokens' min={0} />
            </Col>
            <Col span={12}>
              <Form.InputNumber field='weekly_request_limit' label={t('Request Count')} min={0} />
            </Col>
          </Row>

          <div className='mt-4'>
            <Tag color='red'>{t('Monthly Limits')}</Tag>
          </div>
          <Row gutter={12} className='mt-2'>
            <Col span={12}>
              <Form.InputNumber field='monthly_token_limit' label='Tokens' min={0} />
            </Col>
            <Col span={12}>
              <Form.InputNumber field='monthly_request_limit' label={t('Request Count')} min={0} />
            </Col>
          </Row>
        </Form>
      </div>
    </SideSheet>
  );
};

export default AddLimitModal;
