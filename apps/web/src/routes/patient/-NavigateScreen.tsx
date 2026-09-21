import { useState } from 'react'
import { LanguageToggle, useLocale, useT } from '@/i18n'
import { LargeTextToggle } from '@/preferences'
import {
  AppBar,
  Button,
  ChevronRightIcon,
  Divider,
  FloorPlanMap,
  type FloorPlanRoute,
  InfoNote,
  LinkButton,
  Screen,
  ScreenDock,
} from '@/design-system'
import type { DestinationPlan, NavigatePlan } from '@/features/floorplan'

/**
 * Patient · navigate to a service point — the real floor-plan asset with the
 * destination room highlighted and named (#25), plus, when the current
 * location is known, the walking line and turn-by-turn cues from the
 * navigation API (#28) drawn as a live overlay (#29). Without a location
 * the screen keeps its honest destination-only view — nothing about the
 * route is invented (DESIGN.md). Cross-floor routes open on the floor the
 * patient stands on, with chips to flip floors; the map itself owns pan and
 * zoom. The plan always comes from the caller: the live route computes it
 * from the journey, /design passes its static reference plan.
 */
export function NavigateScreen({
  plan,
  planSvg,
  planUnavailable,
  floorLabel,
  route,
  currentLocation,
  assumedLocation,
  onFloorChange,
  accessibleOnly,
  onToggleAccessibleOnly,
  onScan,
  onPickLocation,
  onBack,
}: {
  plan: NavigatePlan
  /** The displayed floor's drawing. Undefined while it is still being
   *  fetched (ADR-0015 — plans are served, not bundled); the screen keeps
   *  the destination and the turn-by-turn cues, which come from the journey
   *  and the route API, and shows the map card as a placeholder. */
  planSvg?: string
  /** True when the drawing could not be fetched at all, so the placeholder
   *  says so instead of implying it is still coming. */
  planUnavailable?: boolean
  /** Patient-facing floor label, e.g. "Floor 1". The floor list is server
   *  data now, so the screen is handed the lookup rather than doing it. */
  floorLabel: (floorId: string) => string
  route?: { floorId: string; floors: string[]; mapRoute: FloorPlanRoute; cues: string[] }
  /** Plain line stating where the patient is (#35); absent when unknown. */
  currentLocation?: string
  /** The no-fix-yet fallback line — an assumption from the last finished step,
   *  worded so it never claims to be a location (DESIGN.md's honesty rule). */
  assumedLocation?: string
  /** Flips the displayed floor of a cross-floor route (#35 follow-up). */
  onFloorChange?: (floorId: string) => void
  /** Avoid-stairs routing (#99, FR-20) — present renders the toggle chip. */
  accessibleOnly?: boolean
  /** Flips accessibleOnly; the route refetches via the lift on its own. */
  onToggleAccessibleOnly?: () => void
  /** Opens the QR scanner overlay — the screen's single primary action. */
  onScan?: () => void
  /** Opens the same overlay straight at the manual place list. */
  onPickLocation?: () => void
  onBack?: () => void
}) {
  const t = useT()
  const { locale } = useLocale()

  // The displayed floor: the route's choice (origin floor by default, the
  // patient's pick when they flip) or the destination's when routeless.
  const planFloorId = plan.state === 'plan' ? plan.floorId : undefined
  const floorId = route?.floorId ?? planFloorId
  const showFloorSwitch = !!route && route.floors.length > 1 && !!onFloorChange

  return (
    <Screen variant="patient">
      <AppBar
        onBack={onBack}
        backLabel={t('navigate.back')}
        title={appBarTitle(plan, t)}
        subtitle={plan.state === 'plan' ? plan.subtitle : undefined}
        trailing={
          <div className="flex items-center gap-2">
            <LargeTextToggle />
            <LanguageToggle />
          </div>
        }
      />

      {plan.state === 'plan' && floorId ? (
        <>
          {/* The inner relative box is the map card's own frame — chips
              anchored here stay inside the card, not the screen gutter. */}
          <div className="flex-1 overflow-hidden px-gutter pt-1 pb-2">
            <div className="relative h-full w-full">
              {planSvg ? (
                <FloorPlanMap
                  svg={planSvg}
                  floorLabel={floorLabel(floorId)}
                  destination={
                    plan.floorId === floorId
                      ? {
                          placeId: plan.placeId,
                          name: plan.name,
                          x: plan.x,
                          y: plan.y,
                        }
                      : undefined
                  }
                  route={route?.mapRoute}
                  ariaLabel={
                    route
                      ? t('map.routeAria', { floor: plan.floorLabel, name: plan.name })
                      : t('map.planAria', { floor: plan.floorLabel, name: plan.name })
                  }
                  labels={{
                    viewFullFloor: t('map.viewFullFloor'),
                    viewRoute: t('map.viewRoute'),
                    viewDestination: t('map.viewDestination'),
                    youAreHere: t('map.youAreHere'),
                    zoomIn: t('map.zoomIn'),
                    zoomOut: t('map.zoomOut'),
                  }}
                />
              ) : (
                /* The drawing is fetched now, so it can be slow or missing
                   while everything else on this screen is ready. The cues
                   below come from the route API, not from the plan, so the
                   patient can still be walked there by text — never a dead
                   screen waiting on an image. */
                <div className="flex h-full w-full items-center justify-center rounded-lg border border-line bg-surface px-gutter">
                  <p className="text-center font-sans text-caption text-ink-muted">
                    {planUnavailable ? t('map.unavailable') : t('map.loading')}
                  </p>
                </div>
              )}
              {showFloorSwitch && (
                <div className="absolute top-2.5 left-2.5 z-10 flex gap-1.5">
                  {route.floors.map((floor) => (
                    <button
                      key={floor}
                      type="button"
                      onClick={() => onFloorChange?.(floor)}
                      className={`cursor-pointer rounded-full border px-3 py-1.5 font-sans text-caption font-bold ${
                        floor === floorId
                          ? 'border-primary bg-primary-tint text-ink'
                          : 'border-line bg-surface text-ink-muted'
                      }`}
                    >
                      {floorLabel(floor)}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          <ScreenDock>
            <DestinationPanel
              plan={plan}
              cues={route?.cues}
              currentLocation={currentLocation}
              assumedLocation={assumedLocation}
              accessibleOnly={accessibleOnly}
              onToggleAccessibleOnly={onToggleAccessibleOnly}
              onScan={onScan}
              onPickLocation={onPickLocation}
            />
          </ScreenDock>
        </>
      ) : (
        <div className="px-gutter pt-5">
          <InfoNote>{noticeFor(plan, t)}</InfoNote>
        </div>
      )}
    </Screen>
  )
}

type Translate = ReturnType<typeof useT>

function appBarTitle(plan: NavigatePlan, t: Translate): string {
  switch (plan.state) {
    case 'pending':
      return t('navigate.loadingTitle')
    case 'no-destination':
      return t('navigate.noDestinationTitle')
    default:
      return plan.title
  }
}

function noticeFor(plan: NavigatePlan, t: Translate): string {
  switch (plan.state) {
    case 'pending':
      return t('navigate.pendingNotice')
    case 'no-destination':
      return t('navigate.noDestinationNotice')
    default:
      return t('navigate.unsupportedNotice')
  }
}

/**
 * The sheet under the map carries the destination facts patients need —
 * name, floor, place — and, once the current location is known, the
 * turn-by-turn cues derived from the route. It collapses to a one-line peek
 * (destination + floor) so the map keeps the screen while walking; one tap
 * on the peek opens the full body with the cues and the scan actions (the
 * one primary action of this screen). The location block states a real fix
 * in the secondary colour; before any fix, the assumed line says plainly
 * that it is an estimate.
 */
function DestinationPanel({
  plan,
  cues,
  currentLocation,
  assumedLocation,
  accessibleOnly,
  onToggleAccessibleOnly,
  onScan,
  onPickLocation,
}: {
  plan: DestinationPlan
  cues?: string[]
  currentLocation?: string
  assumedLocation?: string
  accessibleOnly?: boolean
  onToggleAccessibleOnly?: () => void
  onScan?: () => void
  onPickLocation?: () => void
}) {
  const t = useT()
  // Always start peeking: the map is the thing the patient is walking by,
  // and the route (assumed or scanned) is usually already drawn. One tap
  // opens the cues and actions.
  const [open, setOpen] = useState(false)

  return (
    <div className="flex flex-col rounded-t-xl bg-surface px-gutter pt-3 shadow-sheet">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        className="flex cursor-pointer flex-col gap-3 bg-none p-0 text-inherit"
      >
        <span className="mx-auto h-1 w-9 rounded-full bg-line" aria-hidden="true" />
        <span className="flex items-baseline justify-between gap-3 pb-3">
          <span className="min-w-0 text-left">
            <span className="block font-sans text-body-md font-bold text-ink">{plan.name}</span>
            <span className="block truncate font-sans text-body-sm text-ink-muted">
              {plan.subtitle}
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-2">
            <span className="rounded-full border border-line bg-neutral px-2.5 py-1 font-sans text-caption font-bold text-ink-muted">
              {plan.floorLabel}
            </span>
            <ChevronRightIcon
              className={`text-ink-muted transition-transform ${open ? 'rotate-90' : '-rotate-90'}`}
            />
          </span>
        </span>
      </button>

      <div
        className={`grid transition-[grid-template-rows] duration-300 ease-out ${open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'}`}
      >
        <div className="min-h-0 overflow-hidden">
          <div className="flex flex-col gap-3.5 pb-6">
            <Divider />
            {/* Avoid-stairs routing (#99): a preference chip, not a modal —
                flipping it refetches the route (the flag rides the query
                key) and the line redraws via the elevator by itself. */}
            {onToggleAccessibleOnly && (
              <button
                type="button"
                aria-pressed={accessibleOnly ?? false}
                onClick={onToggleAccessibleOnly}
                className={`self-start cursor-pointer rounded-full border px-3.5 py-1.5 font-sans text-caption font-bold transition-colors ${
                  accessibleOnly
                    ? 'border-primary bg-primary-tint text-ink'
                    : 'border-line bg-surface text-ink-muted'
                }`}
              >
                {t('navigate.avoidStairs')}
              </button>
            )}
            {/* Where the patient stands (#35) — stated once, plainly; the
                fallback below says so when it is not yet known. */}
            {currentLocation && (
              <p className="m-0 font-sans text-caption font-bold text-secondary">
                {currentLocation}
              </p>
            )}
            {!currentLocation && assumedLocation && (
              <p className="m-0 font-sans text-caption text-ink-muted">{assumedLocation}</p>
            )}
            {cues ? (
              <ol className="m-0 flex list-decimal flex-col gap-1.5 pl-5">
                {cues.map((cue) => (
                  <li key={cue} className="font-sans text-body-sm text-ink">
                    {cue}
                  </li>
                ))}
              </ol>
            ) : null}
            {onScan && !cues && (
              <div className="flex flex-col gap-2">
                <Button block onClick={onScan}>
                  {t('navigate.scanCta')}
                </Button>
                {onPickLocation && (
                  <LinkButton onClick={onPickLocation} className="self-center">
                    {t('navigate.pickInstead')}
                  </LinkButton>
                )}
              </div>
            )}
            {onScan && cues && (
              <LinkButton onClick={onScan} className="self-center">
                {t('navigate.updateLocation')}
              </LinkButton>
            )}
            {!onScan && !cues && (
              <p className="m-0 font-sans text-caption text-ink-muted">
                {t('navigate.waitingLocation')}
              </p>
            )}
            <LinkButton>{t('navigate.askStaffIfLost')}</LinkButton>
          </div>
        </div>
      </div>
    </div>
  )
}
