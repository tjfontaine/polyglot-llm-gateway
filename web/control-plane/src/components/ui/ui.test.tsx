import { describe, it, expect } from 'vitest';
import { render, screen } from '../../test/test-utils';
import { Activity, Database, Route } from 'lucide-react';
import {
  Pill,
  InfoCard,
  OverviewCard,
  Section,
  PageHeader,
  EmptyState,
  LoadingState,
  StatusBadge,
} from './index';

describe('Pill', () => {
  it('renders with icon and label', () => {
    render(<Pill icon={Activity} label="Test Label" />);
    expect(screen.getByText('Test Label')).toBeInTheDocument();
  });

  it('applies correct tone styles', () => {
    const { container } = render(<Pill icon={Activity} label="Amber" tone="amber" />);
    expect(container.firstChild).toHaveClass('border-amber-400/50');
  });

  it('defaults to slate tone', () => {
    const { container } = render(<Pill icon={Activity} label="Default" />);
    expect(container.firstChild).toHaveClass('border-slate-700/70');
  });
});

describe('InfoCard', () => {
  it('renders title, value, and hint', () => {
    render(<InfoCard title="Memory" value="15.0 MB" hint="allocated" icon={Database} />);
    expect(screen.getByText('Memory')).toBeInTheDocument();
    expect(screen.getByText('15.0 MB')).toBeInTheDocument();
    expect(screen.getByText('allocated')).toBeInTheDocument();
  });

  it('renders without hint', () => {
    render(<InfoCard title="Count" value="42" icon={Activity} />);
    expect(screen.getByText('Count')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });
});

describe('OverviewCard', () => {
  it('renders title and subtitle', () => {
    render(
      <OverviewCard
        title="Test Card"
        subtitle="Card subtitle"
        icon={Route}
      />
    );
    expect(screen.getByText('Test Card')).toBeInTheDocument();
    expect(screen.getByText('Card subtitle')).toBeInTheDocument();
  });

  it('renders stats when provided', () => {
    render(
      <OverviewCard
        title="Stats Card"
        subtitle="With stats"
        icon={Route}
        stats={[
          { label: 'Items', value: 10 },
          { label: 'Active', value: '5' },
        ]}
      />
    );
    expect(screen.getByText('Items')).toBeInTheDocument();
    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('renders children', () => {
    render(
      <OverviewCard title="Parent" subtitle="Card" icon={Route}>
        <div data-testid="child">Child content</div>
      </OverviewCard>
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });

  it('renders as link when href provided', () => {
    const { container } = render(
      <OverviewCard title="Link Card" subtitle="Clickable" icon={Route} href="/test" />
    );
    expect(container.querySelector('a[href="/test"]')).toBeInTheDocument();
  });

  it('renders as button when onClick provided', () => {
    const { container } = render(
      <OverviewCard title="Button Card" subtitle="Clickable" icon={Route} onClick={() => {}} />
    );
    expect(container.querySelector('button')).toBeInTheDocument();
  });
});

describe('Section', () => {
  it('renders children with default styling', () => {
    render(
      <Section>
        <p>Section content</p>
      </Section>
    );
    expect(screen.getByText('Section content')).toBeInTheDocument();
  });

  it('applies additional className', () => {
    const { container } = render(
      <Section className="custom-class">
        <p>Content</p>
      </Section>
    );
    expect(container.firstChild).toHaveClass('custom-class');
  });
});

describe('PageHeader', () => {
  it('renders title and subtitle', () => {
    render(<PageHeader title="Page Title" subtitle="Page description" icon={Route} />);
    expect(screen.getByText('Page Title')).toBeInTheDocument();
    expect(screen.getByText('Page description')).toBeInTheDocument();
  });

  it('renders without subtitle', () => {
    render(<PageHeader title="Title Only" icon={Route} />);
    expect(screen.getByText('Title Only')).toBeInTheDocument();
  });

  it('renders actions when provided', () => {
    render(
      <PageHeader
        title="With Actions"
        icon={Route}
        actions={<button>Action Button</button>}
      />
    );
    expect(screen.getByRole('button', { name: 'Action Button' })).toBeInTheDocument();
  });
});

describe('EmptyState', () => {
  it('renders title and description', () => {
    render(
      <EmptyState
        icon={Database}
        title="No data"
        description="There is nothing to show"
      />
    );
    expect(screen.getByText('No data')).toBeInTheDocument();
    expect(screen.getByText('There is nothing to show')).toBeInTheDocument();
  });

  it('renders without description', () => {
    render(<EmptyState icon={Database} title="Empty" />);
    expect(screen.getByText('Empty')).toBeInTheDocument();
  });
});

describe('LoadingState', () => {
  it('renders default message', () => {
    render(<LoadingState />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders custom message', () => {
    render(<LoadingState message="Fetching data..." />);
    expect(screen.getByText('Fetching data...')).toBeInTheDocument();
  });
});

describe('StatusBadge', () => {
  it('renders completed status', () => {
    render(<StatusBadge status="completed" />);
    const badge = screen.getByText('completed');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('bg-emerald-500/20');
  });

  it('renders failed status', () => {
    render(<StatusBadge status="failed" />);
    const badge = screen.getByText('failed');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('bg-red-500/20');
  });

  it('renders cancelled status', () => {
    render(<StatusBadge status="cancelled" />);
    const badge = screen.getByText('cancelled');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('bg-slate-500/20');
  });

  it('renders in_progress status', () => {
    render(<StatusBadge status="in_progress" />);
    const badge = screen.getByText('in_progress');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('bg-amber-500/20');
  });

  it('defaults to in_progress style for unknown status', () => {
    render(<StatusBadge status="unknown" />);
    const badge = screen.getByText('unknown');
    expect(badge).toHaveClass('bg-amber-500/20');
  });
});
