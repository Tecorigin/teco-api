import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Col, Form, Row, SideSheet, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconClose, IconSave } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Title } = Typography;

const EditLimitModal = ({ visible, handleClose, refresh, editingRow }) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [userOptions, setUserOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const isMobile = useIsMobile();

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

    if (editingRow && formApiRef.current) {
      formApiRef.current.setValues({
        user_id: String(editingRow.user_id),
        model_name: editingRow.model_name,
        daily_token_limit: editingRow.daily_token_limit,
        daily_request_limit: editingRow.daily_request_limit,
        weekly_token_limit: editingRow.weekly_token_limit,
        weekly_request_limit: editingRow.weekly_request_limit,
        monthly_token_limit: editingRow.monthly_token_limit,
        monthly_request_limit: editingRow.monthly_request_limit,
      });
    }
  }, [visible, editingRow]);

  const submit = async (values) => {
    if (!editingRow) return;
    const payload = {
      ...values,
      user_id: Number(values.user_id),
    };
    setLoading(true);
    const res = await API.put(`/api/user_model_usage_limit/${editingRow.id}`, payload);
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('Limit configuration updated successfully!'));
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
          <Tag color='blue' shape='circle'>{t('Edit')}</Tag>
          <Title heading={4} className='m-0'>{t('Edit')}</Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={isMobile ? '100%' : 600}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('Save')}
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
                optionList={userOptions}
                placeholder={t('Select existing user')}
                disabled
              />
            </Col>
            <Col span={12}>
              <Form.Select
                field='model_name'
                label={t('Model Name')}
                optionList={modelOptions}
                placeholder={t('Select existing model')}
                disabled
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

export default EditLimitModal;
