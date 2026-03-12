import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { App, Button, Card, Form, Input, Typography } from 'antd';
import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { login } from '../api/client';
import { useAuthStore } from '../store/authStore';

const { Title } = Typography;

interface LoginForm {
  email: string;
  password: string;
}

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const { message } = App.useApp();
  const navigate = useNavigate();
  const location = useLocation();
  const setAuth = useAuthStore((s) => s.setAuth);
  const redirectTo = (location.state as { from?: string } | null)?.from || '/';

  const onFinish = async (values: LoginForm) => {
    setLoading(true);
    try {
      const result = await login(values.email, values.password);

      if (!result.data.is_admin) {
        message.error('Access denied: admin privileges required');
        return;
      }

      const { tokens } = result;
      if (!tokens) {
        throw new Error('Login response is missing tokens');
      }

      setAuth(tokens.access_token, tokens.refresh_token, result.data);
      message.success('Login successful');
      navigate(redirectTo, { replace: true });
    } catch (err) {
      if (err && typeof err === 'object' && 'response' in err) {
        const response = (err as { response: { status: number } }).response;
        if (response.status === 401) {
          message.error('Invalid email or password');
        } else if (response.status === 429) {
          message.error('Too many login attempts. Please try again later.');
        } else {
          message.error('Login failed. Please try again.');
        }
      } else {
        message.error('Network error. Please check your connection.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: '#f0f2f5',
      }}
    >
      <Card style={{ width: 400 }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={3}>CSG Admin</Title>
          <Typography.Text type="secondary">City Stories Guide — Admin Panel</Typography.Text>
        </div>
        <Form<LoginForm>
          name="login"
          onFinish={onFinish}
          autoComplete="off"
          layout="vertical"
          size="large"
        >
          <Form.Item
            name="email"
            rules={[
              { required: true, message: 'Please enter your email' },
              { type: 'email', message: 'Please enter a valid email' },
            ]}
          >
            <Input prefix={<UserOutlined />} placeholder="Email" />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: 'Please enter your password' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="Password" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              Log in
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
