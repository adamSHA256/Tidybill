import {
  Title,
  Text,
  Stack,
  Group,
  Button,
  Textarea,
  Loader,
  Center,
  Paper,
  Collapse,
  CopyButton,
  ActionIcon,
  Tooltip,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { IconArrowLeft, IconDeviceFloppy, IconEye, IconCopy, IconCheck, IconRobot, IconRefresh } from '@tabler/icons-react'
import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api, openTemplatePreview } from '../api/client'
import { useT } from '../i18n'

export function TemplateEditor() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useT()
  const [yamlSource, setYamlSource] = useState('')
  const [originalSource, setOriginalSource] = useState('')
  const [aiPromptOpen, setAiPromptOpen] = useState(false)
  const [generating, setGenerating] = useState(false)

  const { data: template, isLoading: templateLoading } = useQuery({
    queryKey: ['template', id],
    queryFn: () => api.getTemplate(id!),
    enabled: !!id,
  })

  const { data: sourceData, isLoading: sourceLoading } = useQuery({
    queryKey: ['template-source', id],
    queryFn: () => api.getTemplateSource(id!),
    enabled: !!id,
  })

  const { data: aiPromptData } = useQuery({
    queryKey: ['ai-prompt'],
    queryFn: api.getAIPrompt,
    enabled: aiPromptOpen,
  })

  useEffect(() => {
    if (sourceData?.yaml_source) {
      setYamlSource(sourceData.yaml_source)
      setOriginalSource(sourceData.yaml_source)
    }
  }, [sourceData])

  const saveMutation = useMutation({
    mutationFn: () => api.updateTemplateSource(id!, yamlSource),
    onSuccess: () => {
      setOriginalSource(yamlSource)
      queryClient.invalidateQueries({ queryKey: ['template-source', id] })
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      notifications.show({ title: t('template.saved'), message: '', color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const handlePreview = async () => {
    setGenerating(true)
    try {
      // Save first, then generate preview
      if (yamlSource !== originalSource) {
        await api.updateTemplateSource(id!, yamlSource)
        setOriginalSource(yamlSource)
        queryClient.invalidateQueries({ queryKey: ['template-source', id] })
      }
      const data = await api.generatePreview(id!)
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      await openTemplatePreview(id!, data.path)
    } catch (err) {
      notifications.show({ title: t('common.error'), message: (err as Error).message, color: 'red' })
    } finally {
      setGenerating(false)
    }
  }

  const handleReset = () => {
    setYamlSource(originalSource)
  }

  if (templateLoading || sourceLoading) {
    return <Center h={300}><Loader /></Center>
  }

  if (!template) {
    return <Center h={300}><Text>{t('template.not_found')}</Text></Center>
  }

  if (template.is_builtin) {
    return (
      <Center h={300}>
        <Stack align="center">
          <Text>{t('template.builtin_readonly')}</Text>
          <Button variant="light" onClick={() => navigate('/templates')}>
            {t('common.back')}
          </Button>
        </Stack>
      </Center>
    )
  }

  const hasChanges = yamlSource !== originalSource

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Group>
          <Button
            variant="subtle"
            leftSection={<IconArrowLeft size={16} />}
            onClick={() => navigate('/templates')}
          >
            {t('template.back_to_templates')}
          </Button>
          <Title order={3}>{template.name}</Title>
        </Group>
        <Group>
          <Button
            variant="light"
            leftSection={<IconRefresh size={16} />}
            onClick={handleReset}
            disabled={!hasChanges}
          >
            {t('template.reset')}
          </Button>
          <Button
            variant="light"
            color="teal"
            leftSection={<IconEye size={16} />}
            onClick={handlePreview}
            loading={generating}
          >
            {t('template.generate_preview')}
          </Button>
          <Button
            leftSection={<IconDeviceFloppy size={16} />}
            onClick={() => saveMutation.mutate()}
            loading={saveMutation.isPending}
            disabled={!hasChanges}
          >
            {t('common.save')}
          </Button>
        </Group>
      </Group>

      <Textarea
        value={yamlSource}
        onChange={(e) => setYamlSource(e.target.value)}
        autosize
        minRows={20}
        maxRows={50}
        styles={{
          input: {
            fontFamily: 'monospace',
            fontSize: '13px',
            lineHeight: 1.5,
          },
        }}
      />

      <Paper p="sm" withBorder>
        <Group
          justify="space-between"
          onClick={() => setAiPromptOpen(!aiPromptOpen)}
          style={{ cursor: 'pointer' }}
        >
          <Group gap="xs">
            <IconRobot size={18} />
            <Text fw={500}>{t('template.ai_prompt')}</Text>
          </Group>
          <Text size="sm" c="dimmed">
            {aiPromptOpen ? t('common.collapse') : t('common.expand')}
          </Text>
        </Group>

        <Collapse in={aiPromptOpen}>
          <Stack gap="xs" mt="sm">
            <Text size="sm" c="dimmed">{t('template.ai_prompt_desc')}</Text>
            {aiPromptData?.prompt && (
              <>
                <Textarea
                  value={aiPromptData.prompt}
                  readOnly
                  autosize
                  minRows={5}
                  maxRows={15}
                  styles={{
                    input: {
                      fontFamily: 'monospace',
                      fontSize: '12px',
                      lineHeight: 1.4,
                    },
                  }}
                />
                <Group justify="flex-end">
                  <CopyButton value={aiPromptData.prompt}>
                    {({ copied, copy }) => (
                      <Tooltip label={copied ? t('common.copied') : t('common.copy')} events={{ hover: true, focus: true, touch: true }}>
                        <ActionIcon variant="light" onClick={copy}>
                          {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                        </ActionIcon>
                      </Tooltip>
                    )}
                  </CopyButton>
                </Group>
              </>
            )}
          </Stack>
        </Collapse>
      </Paper>
    </Stack>
  )
}
