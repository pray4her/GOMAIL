package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
	"log"
)

// DispatchPlan represents the calculated distribution of emails across various senders.
type DispatchPlan struct {
	// Assignments maps an AccountSenderID to the number of emails it should send.
	Assignments map[int64]int
	// TotalEmails is the total number of emails that will be sent according to this plan.
	TotalEmails int
	// Possible is true if the system can fulfill the requested number of emails with available quotas.
	Possible bool
}

// senderWithQuota is a helper struct to hold a sender and its remaining daily quota.
type senderWithQuota struct {
	sender      model.AccountSender
	dailyRemain int
}

// New struct for the new algorithm
type dispatchCandidate struct {
	sender                model.AccountSender
	dailyRemain           int
	effectiveContribution float64
}

// LoadBalancerService is responsible for calculating the optimal distribution of email sending tasks.
type LoadBalancerService interface {
	GenerateDispatchPlan(userID int64, totalEmailsToSend int) (*DispatchPlan, error)
}

type loadBalancerService struct {
	senderRepo   repository.SenderRepository
	statsRepo    repository.SendStatisticsRepository
	userPermRepo repository.UserPermissionRepository
}

// NewLoadBalancerService creates a new instance of LoadBalancerService.
func NewLoadBalancerService(
	senderRepo repository.SenderRepository,
	statsRepo repository.SendStatisticsRepository,
	userPermRepo repository.UserPermissionRepository,
) LoadBalancerService {
	return &loadBalancerService{
		senderRepo:   senderRepo,
		statsRepo:    statsRepo,
		userPermRepo: userPermRepo,
	}
}

// GenerateDispatchPlan creates a sending plan based on user permissions, sender weights, and daily limits.
func (s *loadBalancerService) GenerateDispatchPlan(userID int64, totalEmailsToSend int) (*DispatchPlan, error) {
	// 1. Get all senders the user is allowed to use.
	allowedSenderIDs, err := s.userPermRepo.FindAllowedAccountSenderIDs(userID)
	if err != nil {
		return nil, err
	}
	if len(allowedSenderIDs) == 0 {
		return &DispatchPlan{Possible: false}, nil
	}

	// 2. Fetch full details for these senders.
	senders, err := s.senderRepo.FindAccountSenderDetailsByIDs(allowedSenderIDs)
	if err != nil {
		return nil, err
	}

	// 3. Get today's sent counts for these senders.
	sentCounts, err := s.statsRepo.GetTotalSentCountForToday(allowedSenderIDs)
	if err != nil {
		return nil, err
	}

	// 4. Organize senders by account and calculate remaining quotas.
	accounts := make(map[int64][]senderWithQuota)
	totalSystemQuota := 0
	for _, sender := range senders {
		if sender.Status != "active" || sender.Account.Status != "active" {
			continue // Skip inactive senders or accounts
		}
		sentToday := sentCounts[sender.ID] // Defaults to 0 if not in map
		senderRemain := sender.DailySendLimit - sentToday
		if senderRemain > 0 {
			sq := senderWithQuota{sender: sender, dailyRemain: senderRemain}
			accounts[sender.AccountID] = append(accounts[sender.AccountID], sq)
			totalSystemQuota += senderRemain
		}
	}

	// 5. Initial Plan Setup
	plan := &DispatchPlan{
		Assignments: make(map[int64]int),
		Possible:    totalEmailsToSend <= totalSystemQuota,
		TotalEmails: min(totalEmailsToSend, totalSystemQuota),
	}
	if !plan.Possible {
		log.Printf("Warning: Requested %d emails, but only %d quota remaining.", totalEmailsToSend, totalSystemQuota)
	}

	emailsToAssign := plan.TotalEmails
	if emailsToAssign == 0 {
		return plan, nil
	}

	// 6. New "Weighted Quota Distribution" Algorithm
	// 6.1 Prepare candidates and calculate total effective contribution
	var candidates []dispatchCandidate
	var totalEffectiveContribution float64
	for _, senderList := range accounts {
		for _, sq := range senderList {
			contribution := float64(sq.dailyRemain) * float64(sq.sender.Weight)
			if contribution > 0 {
				candidate := dispatchCandidate{
					sender:                sq.sender,
					dailyRemain:           sq.dailyRemain,
					effectiveContribution: contribution,
				}
				candidates = append(candidates, candidate)
				totalEffectiveContribution += contribution
			}
		}
	}

	if totalEffectiveContribution == 0 {
		return plan, nil // No one can send emails
	}

	// 6.2 Proportional Distribution
	totalAssigned := 0
	for _, candidate := range candidates {
		share := (candidate.effectiveContribution / totalEffectiveContribution) * float64(emailsToAssign)
		assigned := min(int(share), candidate.dailyRemain)
		plan.Assignments[candidate.sender.ID] = assigned
		totalAssigned += assigned
	}

	// 6.3 Distribute Remainder
	remainder := emailsToAssign - totalAssigned
	if remainder > 0 {
		// Sort candidates by their remaining capacity to fairly distribute the remainder
		for i := 0; i < remainder; i++ {
			for _, candidate := range candidates {
				if plan.Assignments[candidate.sender.ID] < candidate.dailyRemain {
					plan.Assignments[candidate.sender.ID]++
					remainder--
					if remainder == 0 {
						break
					}
				}
			}
			if remainder == 0 {
				break
			}
		}
	}

	return plan, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getLastAccountKey(m map[int64][]senderWithQuota) int64 {
	var lastKey int64
	for k := range m {
		lastKey = k
	}
	return lastKey
}
